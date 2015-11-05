// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import (
	"sync"
	"testing"
)

type Node struct {
	T *Tracker
}

func (n *Node) RpcRequestRoute(r RpcRequest) {
	if m, ok := r.(*Msg); ok {
		n.T.ResponseRPC(m)
	}
}

func (n *Node) RpcResponseRoute(resp RpcResponse, req RpcRequest) {
}

func (n *Node) DefaultRoute(v interface{}) {
}

func (n *Node) ErrorRoute(v interface{}, text string) {
}

type Msg struct {
	id uint32
}

func (m *Msg) RpcSetId(id uint32) {
	m.id = id
}

func (m *Msg) RpcGetId() uint32 {
	return m.id
}

func TestRPC(t *testing.T) {
	n := new(Node)
	tracker := NewRPCTracker(1, n)
	n.T = tracker

	tracker.Run()

	m1 := new(Msg)
	m2 := new(Msg)

	tracker.RequestRPC(m1)
	tracker.RequestRPC(m2)

	tracker.RequestRPC(m1)

	tracker.Stop()
}

func BenchmarkTracker(b *testing.B) {
	var tracker *Tracker
	n := new(Node)
	if tracker = NewRPCTracker(16384, n); tracker == nil {
		b.FailNow()
	}
	n.T = tracker
	tracker.Run()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m1 := new(Msg)
			tracker.RequestRPC(m1)
		}
	})

}

func BenchmarkMutex(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var m sync.Mutex
		for pb.Next() {
			m.Lock()
			m.Unlock()
		}
	})
}

func BenchmarkChan(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		c := make(chan uint64, 16384)
		for pb.Next() {
			select {
			case c <- 1:
			case <-c:
			}
		}
	})
}
