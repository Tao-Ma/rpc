// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

var (
	errQuit      error = &Error{err: "quit"}
	errShortRead error = &Error{err: "short read"}
)

// IOChannel
type IOChannel interface {
	In() chan Payload
	Out() chan Payload

	// Support decorate data
	Unwrap(Payload) Payload
	Wrap(Payload) Payload

	InError(error)
	OutError(error)
}

/*
type Encoder interface {
	GetHdrLen() uint32
	GetPayloadLen() uint32

	MarshalHeader([]byte, Payload, uint32) error
	MarshalPayload(Payload, []byte) ([]byte, error)
}

type Decoder interface {
	GetHdrLen() uint32
	GetPayloadLen() uint32

	UnmarshalHeader([]byte) error
	UnmarshalPayload([]byte) (Payload, error)
}
*/

type MsgFactory interface {
	NewBuffer() MsgBuffer
}

type MsgBuffer interface {
	Reset()

	MarshalHeader([]byte, Payload, uint32) error
	UnmarshalHeader([]byte) error

	MarshalPayload(Payload, []byte) ([]byte, error)
	UnmarshalPayload([]byte) (Payload, error)

	SetPayloadInfo(Payload)
	GetPayloadInfo(Payload)

	GetHdrLen() uint32
	GetPayloadLen() uint32
}
