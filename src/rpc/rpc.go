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
	RpcResponse
}

type RpcResponse interface {
	RpcGetId() uint32
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
	id   chan uint32

	timeout uint32
	init    chan RpcRequest
	done    chan RpcResponse

	rpcs map[uint32]*call
}

func NewRPCTracker(n uint32) *Tracker {
	t := new(Tracker)

	t.quit = make(chan bool)

	if n < 1 || n > 4096 {
		return nil
	}

	t.n = n
	t.next = 1
	t.id = make(chan uint32, t.n)

	t.init = make(chan RpcRequest)
	t.done = make(chan RpcResponse)
	t.rpcs = make(map[uint32]*call)

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
	case id := <-t.id:
		r.RpcSetId(id)
		select {
		case t.init <- r:
		case <-time.Tick(time.Second):
			return
		}
	case <-time.Tick(time.Second):
		// TODO: tell the caller prepare failed(timeout).
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
		case <-time.Tick(time.Second):
			// TODO timeout?
		}
	}
}

func (t *Tracker) AbortRPC(v interface{}) {
	if r, ok := v.(RpcRequest); !ok {
		// do something
	} else {
		select {
		case t.done <- r:
		case <-time.Tick(time.Second):
			// TODO timeout?
		}
	}
}

func (t *Tracker) loop() {
	for i := uint32(0); i < t.n; i++ {
		select {
		case t.id <- t.next:
			t.next++
		}
	}

forever:
	for {
		select {
		case <-t.quit:
			break forever
		case r := <-t.init:
			if _, exist := t.rpcs[r.RpcGetId()]; exist {
				// TODO: add multiple times?
			} else {
				c := new(call)
				c.id = r.RpcGetId()
				c.req = r
				t.rpcs[r.RpcGetId()] = c
			}
		case r := <-t.done:
			if c, exist := t.rpcs[r.RpcGetId()]; exist {
				c.resp = r
				delete(t.rpcs, r.RpcGetId())
				select {
				case t.id <- t.next:
					t.next++
				default:
					// TODO: should not happen?
				}
			} else {
				// TODO: timeout or unexpected(invalid?)
			}
		case <-time.Tick(time.Second):
			// need a data structure to maintain recently timeout rpc
		}
	}
}
