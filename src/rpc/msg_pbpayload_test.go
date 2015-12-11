// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
	"testing"
)

func TestProtobufFactory(t *testing.T) {
	pf := NewMsgProtobufFactory()
	pbf1 := pf.NewBuffer()
	b1 := make([]byte, 4096)

	if ResourceReqID != 1 {
		t.Fail()
	}

	d1 := NewResourceReq()
	d1.Id = proto.Uint64(10086)

	b3, err := pbf1.Marshal(d1, b1)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	d2, err := pbf1.Unmarshal(ResourceReqID, b3)
	if err != nil {
		t.Log(len(b1))
		t.FailNow()
	}

	if d1.GetMsgPayloadID() != d2.GetMsgPayloadID() {
		t.Fail()
	}

	d3 := d2.(*ResourceReq)

	if d1.GetId() != d3.GetId() {
		t.Fail()
	}
}
