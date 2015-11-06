// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

/*
 * Track RPC
 *  RequestRPC()
 *  ResponseRPC()
 *  AbortRPC()
 */
package rpc

import (
	"time"
)

type RpcRequest interface {
	RpcSetId(uint32)
}

type RpcResponse interface {
	RpcGetId() uint32
}

type Router interface {
	RpcRequestRoute(RpcRequest)
	RpcResponseRoute(RpcResponse, RpcRequest)

	DefaultRoute(interface{})

	ErrorRoute(interface{}, string)
}

type call struct {
	id   uint32
	req  RpcRequest
	resp RpcResponse
}

type Tracker struct {
	bg   *BackgroudService
	n    uint32
	next uint32

	timeout uint32
	init    chan RpcRequest
	done    chan RpcResponse

	rpcs map[uint32]*call

	r Router

	req_timeout time.Duration
	srv_timeout time.Duration
}

func NewRPCTracker(n uint32, r Router) *Tracker {
	t := new(Tracker)

	if n < 0 || n > 16384 {
		return nil
	}

	if bg, err := NewBackgroundService(t); err != nil {
		return nil
	} else {
		t.bg = bg
	}

	t.n = n
	t.next = 1

	t.init = make(chan RpcRequest, t.n)
	t.done = make(chan RpcResponse, t.n)
	t.rpcs = make(map[uint32]*call)

	t.r = r

	t.req_timeout = 1000 * time.Millisecond
	t.srv_timeout = 000 * time.Millisecond

	return t
}

func (t *Tracker) Run() {
	t.bg.Run()
}

func (t *Tracker) Stop() {
	t.bg.Stop()
}

func (t *Tracker) RequestRPC(v interface{}) {
	var r RpcRequest

	if nr, ok := v.(RpcRequest); !ok {
		t.r.DefaultRoute(v)
		return
	} else {
		r = nr
	}

	select {
	case t.init <- r:
	default:
		select {
		case t.init <- r:
		case <-time.Tick(t.req_timeout):
			// TODO: tell the caller prepare failed(timeout).
		}
		return
	}

	return
}

func (t *Tracker) ResponseRPC(v interface{}) {
	if r, ok := v.(RpcResponse); !ok {
		t.r.DefaultRoute(v)
	} else {
		select {
		case t.done <- r:
		default:
			select {
			case t.done <- r:
			case <-time.Tick(t.req_timeout):
				// TODO timeout?
			}
		}
	}
}

func (t *Tracker) ServiceLoop(q chan bool, r chan bool) {
	r <- true
forever:
	for {
		select {
		case <-q:
			break forever
		case r := <-t.init:
			id := t.next
			t.next++
			r.RpcSetId(id)
			if _, exist := t.rpcs[id]; exist {
				panic("")
			} else {
				c := new(call)
				c.id = id
				c.req = r
				t.rpcs[c.id] = c
				go t.r.RpcRequestRoute(r)
			}
		case r := <-t.done:
			if c, exist := t.rpcs[r.RpcGetId()]; exist {
				c.resp = r
				delete(t.rpcs, c.id)
				go t.r.RpcResponseRoute(c.resp, c.req)
			} else {
				// timeout or unexpected
				go t.r.ErrorRoute(r, "timeout or feature response")
			}
		case <-time.Tick(t.srv_timeout):
			// TODO: move this to a seperate goroutine, timer is hotspot
			// if request is too much
			// need a data structure to maintain recently timeout rpc
		}
	}
}
