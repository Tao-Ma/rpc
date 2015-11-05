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

type RpcProcessor interface {
	Process(interface{})
}

type call struct {
	id   uint32
	req  RpcRequest
	resp RpcResponse
}

type Tracker struct {
	running bool
	quit    chan bool
	err     error

	n    uint32
	next uint32

	timeout uint32
	init    chan RpcRequest
	done    chan RpcResponse

	rpcs map[uint32]*call

	p RpcProcessor

	req_timeout time.Duration
	srv_timeout time.Duration
}

func NewRPCTracker(n uint32, p RpcProcessor) *Tracker {
	t := new(Tracker)

	t.quit = make(chan bool)

	if n < 1 || n > 16384 {
		return nil
	}

	t.n = n
	t.next = 1

	t.init = make(chan RpcRequest, t.n)
	t.done = make(chan RpcResponse, t.n)
	t.rpcs = make(map[uint32]*call)

	t.p = p

	t.req_timeout = 1000 * time.Millisecond
	t.srv_timeout = 000 * time.Millisecond

	return t
}

func (t *Tracker) Run() {
	go t.loop()
}

func (t *Tracker) Stop() {
	if !t.running {
		return
	}

	t.quit <- true
	// cleanup?
	t.running = false
}

func (t *Tracker) RequestRPC(v interface{}) {
	var r RpcRequest

	if nr, ok := v.(RpcRequest); !ok {
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
		// do something
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

func (t *Tracker) loop() {
forever:
	for {
		select {
		case <-t.quit:
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
				go t.p.Process(r)
			}
		case r := <-t.done:
			if c, exist := t.rpcs[r.RpcGetId()]; exist {
				c.resp = r
				delete(t.rpcs, c.id)
				//go something(r.req, r.resp)
			} else {
				// TODO: timeout or unexpected(invalid?)
			}
		case <-time.Tick(t.srv_timeout):
			// TODO: move this to a seperate goroutine, timer is hotspot
			// if request is too much
			// need a data structure to maintain recently timeout rpc
		}
	}
}
