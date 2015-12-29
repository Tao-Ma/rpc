// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"time"
)

var (
	ErrNoAvailableSlot error = &Error{err: "TimeoutTracker no available slot"}
	ErrTimeout         error = &Error{err: "TimeoutTracker Timeout"}
)

// support timeout object
type TrackObject interface {
	When() time.Time
	Timeout(time.Time)
}

type TrackID uint64

const (
	InvalidTrackID = TrackID(0)
)

type TimeoutTracker struct {
	ticker *time.Ticker

	delta time.Duration

	id TrackID

	last time.Time

	idx   map[TrackID]*LinkNode   // fast remove index
	ridx  map[*LinkNode]TrackID   // fast remove index
	tslot map[time.Time]*LinkNode // fast insert

	free *LinkNode
}

func NewTimeoutTracker(ms int, max int) (*TimeoutTracker, error) {
	if ms <= 0 {
		ms = 50
	}

	if max <= 0 {
		max = 1024
	}

	tt := new(TimeoutTracker)

	tt.free = NewLink()
	for i := 0; i < max; i++ {
		tt.free.InsertAfter(tt.free.Tail(), NewLinkNode(nil))
	}

	tt.delta = time.Duration(ms) * time.Millisecond

	tt.id = 1

	// will not timeout again.
	tt.last = time.Now().Truncate(tt.delta)
	tt.ticker = time.NewTicker(tt.delta)

	tt.idx = make(map[TrackID]*LinkNode)
	tt.ridx = make(map[*LinkNode]TrackID)
	tt.tslot = make(map[time.Time]*LinkNode)

	return tt, nil
}

func (tt *TimeoutTracker) Stop() {
	tt.ticker.Stop()
}

func (tt *TimeoutTracker) Tick() <-chan time.Time {
	return tt.ticker.C
}

func timeoutCheckLinkDo(v interface{}, arg LinkDoFuncArg) {
	o := v.(TrackObject)
	o.Timeout(arg.(time.Time))
}

// TimeoutCheck checks the tracked object timeout states. It uses now to
// timeout objects.
func (tt *TimeoutTracker) TimeoutCheck(now time.Time) {
	now = now.Truncate(tt.delta)
	for now.After(tt.last) {
		tt.last = tt.last.Add(tt.delta)
		if l, exist := tt.tslot[tt.last]; exist {
			delete(tt.tslot, now)

			l.Do(timeoutCheckLinkDo, now)
			for !l.Empty() {
				n := l.Remove(l.Head())
				id, exist := tt.ridx[n]
				if !exist {
					panic("TimeoutTracker data inconsistent")
				}
				delete(tt.idx, id)
				delete(tt.ridx, n)
				tt.free.InsertAfter(tt.free.Tail(), n)
			}
			// TODO: reuse Link in the future
			delete(tt.tslot, tt.last)
		}
	}
}

func (tt *TimeoutTracker) Add(o TrackObject) (TrackID, error) {
	t := o.When().Truncate(tt.delta)

	if t.Before(tt.last) {
		o.Timeout(tt.last)
		return InvalidTrackID, ErrTimeout
	}

	if tt.free.Empty() {
		return InvalidTrackID, ErrNoAvailableSlot
	}

	id := tt.id
	tt.id++

	l, exist := tt.tslot[t]
	if !exist {
		// TODO: reuse Link in the future
		tt.tslot[t] = NewLink()
		l = tt.tslot[t]
	}

	tt.idx[id] = tt.free.Head().Remove(tt.free.Head()).Set(o)
	tt.ridx[tt.idx[id]] = id
	l.InsertAfter(l.Tail(), tt.idx[id])

	return id, nil
}

func (tt *TimeoutTracker) Del(id TrackID) TrackObject {
	n, exist := tt.idx[id]
	if !exist {
		return nil
	}

	delete(tt.idx, id)
	delete(tt.ridx, n)
	o := n.v.(TrackObject)

	l, exist := tt.tslot[o.When().Truncate(tt.delta)]
	if !exist {
		panic("TimeoutTracker data inconsistent")
	}

	l.Remove(n)
	tt.free.InsertAfter(tt.free.Tail(), n)

	if l.Empty() {
		// TODO: reuse Link in the future
		delete(tt.tslot, o.When().Truncate(tt.delta))
	}

	return o
}

type LinkNode struct {
	prev *LinkNode
	next *LinkNode

	v interface{}
}

func NewLink() *LinkNode {
	l := NewLinkNode(nil)
	l.next, l.prev = l, l
	return l
}

func NewLinkNode(v interface{}) *LinkNode {
	return new(LinkNode).Set(v)
}

func (n *LinkNode) Set(v interface{}) *LinkNode {
	n.v = v
	return n
}

func (l *LinkNode) Empty() bool {
	return l == l.next
}

func (l *LinkNode) Head() *LinkNode {
	return l.next
}

func (l *LinkNode) Tail() *LinkNode {
	return l.prev
}

func (l *LinkNode) Next(n *LinkNode) *LinkNode {
	if l.Empty() {
		panic("Link Next() on invalid LinkNode")
	}
	if l.Tail() == n {
		return nil
	}
	return n.next
}

func (l *LinkNode) InsertAfter(mark *LinkNode, n *LinkNode) {
	n.next, n.prev = mark.next, mark
	mark.next, n.next.prev = n, n
}

func (l *LinkNode) Remove(n *LinkNode) *LinkNode {
	n.prev.next, n.next.prev = n.next, n.prev
	n.next, n.prev = nil, nil

	return n
}

type LinkDoFunc func(v interface{}, arg LinkDoFuncArg)
type LinkDoFuncArg interface{}

func (l *LinkNode) Do(f LinkDoFunc, arg LinkDoFuncArg) {
	for cur := l.Head(); cur != nil; cur = l.Next(cur) {
		f(cur.v, arg)
	}
}
