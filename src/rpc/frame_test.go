// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
	"io"
	"testing"
	"time"
)

type TestWrapper struct{}

func (tw *TestWrapper) wrap(p Payload) Payload {
	return p
}

func TestMockMsg(t *testing.T) {
	pr, pw := io.Pipe()
	ch := make(chan Payload)
	tw := &TestWrapper{}

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	w := NewWriter(pw, ch, hf, nil)
	r := NewReader(pr, ch, tw, hf, nil)

	w.Run()
	r.Run()

	req := NewResourceReq()
	req.Id = proto.Uint64(10000)
	w.Write(req)
	select {
	case <-time.Tick(time.Second):
	}

	resp := <-ch

	if resp.(*ResourceReq).GetId() != 10000 {
		t.Log(req, resp)
		t.Fail()
	}

	w.Stop()
	r.Stop()
}
