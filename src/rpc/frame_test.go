// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import (
	"github.com/golang/protobuf/proto"
	"io"
	"testing"
	"time"
)

var recv_id int32

type FakeProcessor struct{}

func (p *FakeProcessor) Process(v interface{}) {
	recv_id = 10086
}

func TestMockMsg(t *testing.T) {
	pr, pw := io.Pipe()

	hf := NewDefaultHeaderFactory()
	pf := NewProtobufFactory()

	w := NewWriter(pw, hf, pf)
	r := NewReader(pr, hf, pf, &FakeProcessor{})

	w.Run()
	r.Run()

	req := NewResourceReq()
	req.Id = proto.Int32(10000)
	w.Process(req)
	select {
	case <-time.Tick(time.Second):
	}

	w.Stop()
	r.Stop()

	if recv_id != 10086 {
		t.Log(r)
		t.Log(w)
		t.Fail()
	}
}
