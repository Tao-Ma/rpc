// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

/*
 * Track RPC
 *  RequestRPC()
 *  ResponseRPC()
 *  AbortRPC()
 */
package lrpc

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

type Tracker struct {
	running bool
	quit    chan bool
	err     error

	n    uint32
	next uint32
	id   chan uint32

	timeout uint32
	init    chan uint32
	done    chan uint32

	rpcs map[uint32]uint32
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

	t.init = make(chan uint32)
	t.done = make(chan uint32)
	t.rpcs = make(map[uint32]uint32)

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
		case t.init <- id:
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
		case t.done <- r.RpcGetId():
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
		case t.done <- r.RpcGetId():
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
		case id := <-t.init:
			if _, exist := t.rpcs[id]; exist {
				// TODO: add multiple times?
			} else {
				t.rpcs[id] = t.timeout
			}
		case id := <-t.done:
			if _, exist := t.rpcs[id]; exist {
				delete(t.rpcs, id)
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
