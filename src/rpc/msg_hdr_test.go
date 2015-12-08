// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"testing"
)

func TestHeaderFactory(t *testing.T) {
	/*
		hf := NewDefaultHeaderFactory()
		hbf1 := hf.NewBufferFactory()
		hbf2 := hf.NewBufferFactory()
		b1 := make([]byte, hbf1.GetHdrLen())
		b2 := make([]byte, hbf2.GetHdrLen())

		hbf1.SetPayloadId(1777)
		hbf1.SetPayloadLen(100)

		if err := hbf1.Marshal(b1); err != nil {
			t.FailNow()
		}

		if copy(b2, b1) != 16 {
			t.FailNow()
		}

		if err := hbf2.Unmarshal(b2); err != nil {
			t.FailNow()
		}

		if hbf1.GetPayloadId() != hbf2.GetPayloadId() {
			t.Fail()
		}

		if hbf1.GetPayloadLen() != hbf2.GetPayloadLen() {
			t.Fail()
		}
	*/
}
