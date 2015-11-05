// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import (
	"github.com/golang/protobuf/proto"
	"io"
	"testing"
	"time"
)

var recv_id int32

type node struct {
	T *Tracker
}

func (n *node) RpcRequestRoute(r RpcRequest) {
}

func (n *node) RpcResponseRoute(resp RpcResponse, req RpcRequest) {
}

func (n *node) DefaultRoute(v interface{}) {
	recv_id = 10086
}

func (n *node) ErrorRoute(v interface{}, text string) {
}

func TestMockMsg(t *testing.T) {
	pr, pw := io.Pipe()

	hf := NewDefaultHeaderFactory()
	pf := NewProtobufFactory()
	n := new(node)

	w := NewWriter(pw, hf, pf)
	r := NewReader(pr, hf, pf, n)

	w.Run()
	r.Run()

	req := NewResourceReq()
	req.Id = proto.Int32(10000)
	w.Write(req)
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
