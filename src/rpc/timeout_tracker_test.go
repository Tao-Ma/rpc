// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"testing"
	"time"
)

func TestLink(t *testing.T) {
	l := NewLink()

	if !l.Empty() {
		t.FailNow()
		t.Log("Link is not empty after init()")
	}

	l.InsertAfter(l, NewLinkNode(1))
	l.InsertAfter(l.Tail(), NewLinkNode(2))
	l.InsertAfter(l.Tail(), NewLinkNode(3))

	s := 0
	l.Do(func(v interface{}, arg LinkDoFuncArg) { s += v.(int) }, nil)
	if s != 6 {
		t.Log("Link Do() sum result error")
		t.FailNow()
	}
	n := l.Remove(l.Head())
	if n.v.(int) != 1 {
		t.Log("Link first element error")
		t.FailNow()
	}
	n = l.Remove(l.Head())
	if n.v.(int) != 2 {
		t.Log("Link second element error")
		t.FailNow()
	}
	n = l.Remove(l.Head())
	if n.v.(int) != 3 {
		t.Log("Link third element error")
		t.FailNow()
	}
}

type testTimeoutObject struct {
	n  string
	to time.Time
	t  *testing.T
}

func newTestTimeoutObject(n string, to time.Time, t *testing.T) TrackObject {
	tto := new(testTimeoutObject)
	tto.n = n
	tto.to = to
	tto.t = t
	return TrackObject(tto)
}

func (tto *testTimeoutObject) When() time.Time {
	return tto.to
}

func (tto *testTimeoutObject) Timeout(now time.Time) {
	tto.t.Log(tto.n, " timeout: ", now)
}

func TestTimeoutTracker(t *testing.T) {
	delta := 100
	tt, _ := NewTimeoutTracker(delta, 1024)

	begin := time.Now()
	to_uint := time.Duration(delta) * time.Millisecond

	tto1 := newTestTimeoutObject("1", begin.Add(1*to_uint), t)
	tto2a := newTestTimeoutObject("2a", begin.Add(2*to_uint), t)
	tto2b := newTestTimeoutObject("2b", begin.Add(2*to_uint), t)
	tto3 := newTestTimeoutObject("3", begin.Add(3*to_uint), t)
	tto4a := newTestTimeoutObject("4a", begin.Add(4*to_uint), t)
	tto4b := newTestTimeoutObject("4b", begin.Add(4*to_uint), t)
	tto4c := newTestTimeoutObject("4c", begin.Add(4*to_uint), t)
	tto4d := newTestTimeoutObject("4d", begin.Add(4*to_uint), t)
	tto6 := newTestTimeoutObject("6", begin.Add(6*to_uint), t)

	tt.Add(tto1)
	tt.Add(tto2a)
	tt.Add(tto2b)
	tt.Add(tto3)
	tto4a_id, _ := tt.Add(tto4a)
	tt.Add(tto4b)
	tto4c_id, _ := tt.Add(tto4c)
	tt.Add(tto4d)
	tt.Add(tto6)

	// 1
	now := <-tt.Tick()
	tt.TimeoutCheck(now)
	// dup check should ok
	tt.TimeoutCheck(now)

	// 2
	now = <-tt.Tick()
	tt.TimeoutCheck(now)

	// 3
	now = <-tt.Tick()
	tt.TimeoutCheck(now)

	tt.Del(tto4c_id)
	tt.Del(tto4a_id)

	// 4
	now = <-tt.Tick()
	tt.TimeoutCheck(now)

	// 5
	now = <-tt.Tick()
	tt.TimeoutCheck(now)

	// 6
	now = <-tt.Tick()
	tt.TimeoutCheck(now)

	// 7
	now = <-tt.Tick()
	tt.TimeoutCheck(now)

	tt.Stop()
}
