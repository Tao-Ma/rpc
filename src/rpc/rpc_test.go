// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import (
	"sync"
	"testing"
)

type Msg struct {
	id uint32
}

func (m *Msg) RpcSetId(id uint32) {
	m.id = id
}

func (m *Msg) RpcGetId() uint32 {
	return m.id
}

func (t *Tracker) Process(v interface{}) {
	t.ResponseRPC(v)
}

func (t *Tracker) SetProcessor(p RpcProcessor) {
	t.p = p
}

func TestRPC(t *testing.T) {
	tracker := NewRPCTracker(1, nil)
	tracker.SetProcessor(tracker)

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
	if tracker = NewRPCTracker(16384, nil); tracker == nil {
		b.FailNow()
	}
	tracker.SetProcessor(tracker)
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
		c := make(chan uint64, 100)
		for pb.Next() {
			select {
			case c <- 1:
			case <-c:
			}
		}
	})
}
