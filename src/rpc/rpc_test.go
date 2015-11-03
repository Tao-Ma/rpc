// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import (
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

func TestRPC(t *testing.T) {
	tracker := NewRPCTracker(2)

	tracker.Run()

	m1 := new(Msg)
	m2 := new(Msg)

	tracker.RequestRPC(m1)
	t.Logf("m1:%v", m1)
	tracker.RequestRPC(m2)
	t.Logf("m2:%v", m2)

	tracker.ResponseRPC(m2)
	tracker.ResponseRPC(m1)

	tracker.RequestRPC(m1)
	if m1.id != 3 {
		t.Fail()
	}

	tracker.Stop()
}
