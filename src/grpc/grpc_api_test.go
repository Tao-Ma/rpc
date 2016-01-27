// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package grpc

import (
	"net"
	"testing"
)

func TestGRPC(t *testing.T) {
	addr := ":12345"

	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.FailNow()
	}

	s := NewServer()
	go s.Serve(l)
	defer s.Stop()

	c, err := Dial(addr)
	if err != nil {
		t.FailNow()
	}
	defer c.Close()
}
