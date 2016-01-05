// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
	"io"
	"testing"
	"time"
)

type chanPayload chan Payload

func (io chanPayload) In() chan Payload {
	return io
}
func (io chanPayload) Out() chan Payload {
	return io
}
func (io chanPayload) Wrap(p Payload) Payload {
	return p
}
func (io chanPayload) Unwrap(p Payload) Payload {
	return p
}
func (io chanPayload) InError(err error) {
}
func (io chanPayload) OutError(err error) {
}

func TestMockMsg(t *testing.T) {
	pr, pw := io.Pipe()
	ch := make(chanPayload, 128)

	hf := NewMsgHeaderFactory(NewMsgProtobufFactory())

	w := NewWriter(pw, ch, hf.NewBuffer(), nil)
	r := NewReader(pr, ch, hf.NewBuffer(), nil)

	w.Run()
	r.Run()

	req := NewResourceReq()
	req.Id = proto.Uint64(10000)
	w.Write(req)
	<-time.After(1 * time.Second)

	select {
	case resp := <-ch:
		if resp.(*ResourceReq).GetId() != 10000 {
			t.Log(req, resp)
			t.Fail()
		}
	case <-time.After(1 * time.Second):
		t.Log("timeout")
		t.FailNow()
	}

	r.Stop()
	w.Stop()
}
