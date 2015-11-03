// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package lrpc

import (
	_ "github.com/golang/protobuf/proto"
	"testing"
)

func TestHeaderFactory(t *testing.T) {
	//hf := NewDefaultHeaderFactory()

	h1 := NewHeader()

	if h1.GetLen() != 16 {
		t.Fail()
	}

	h1.SetPayloadId(1777)
	h1.SetPayloadLen(100)

	b, _ := h1.Marshal()

	h2 := NewHeader()
	_ = h2.Unmarshal(b)

	if h1.GetPayloadId() != h2.GetPayloadId() {
		t.Fail()
	}

	if h1.GetPayloadLen() != h2.GetPayloadLen() {
		t.Fail()
	}
}
