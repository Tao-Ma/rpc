// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
	"testing"
)

func TestProtobufFactory(t *testing.T) {
	pf := NewProtobufFactory()

	if ResourceReqId != 1 {
		t.Fail()
	}

	d1 := NewResourceReq()
	d1.Id = proto.Uint64(10086)

	b, _ := d1.MarshalPayload()

	d2 := pf.New(ResourceReqId)
	_ = d2.UnmarshalPayload(b)

	if d1.GetPayloadId() != d2.GetPayloadId() {
		t.Fail()
	}

	d3 := d2.(*ResourceReq)

	if d1.GetId() != d3.GetId() {
		t.Fail()
	}
}
