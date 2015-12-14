// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
)

type protobufFactory struct{}

func NewProtobufFactory() RPCPayloadFactory {
	pf := new(protobufFactory)
	return RPCPayloadFactory(pf)
}

type protobufBuffer struct {
	buf proto.Buffer
}

func (pf *protobufFactory) NewBuffer() RPCPayloadBuffer {
	pb := new(protobufBuffer)

	return RPCPayloadBuffer(pb)
}

func (pb *protobufBuffer) Marshal(p Payload, b []byte) ([]byte, error) {
	// Refer: github.com/golang/protobuf/proto/encode.go
	// func Marshal(pb Message) ([]byte, error)
	m, ok := p.(proto.Message)
	if !ok {
		// TODO: error
		return nil, nil
	}

	pb.buf.SetBuf(b)
	pb.buf.Reset()
	err := pb.buf.Marshal(m)
	if err != nil {
		return nil, err
	}

	return pb.buf.Bytes(), nil
}

func (pb *protobufBuffer) Unmarshal(name string, b []byte) (Payload, error) {
	return b, nil
}
