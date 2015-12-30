// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

// Error
var (
	// client_api
	ErrCallTimeout error = &Error{err: "Call timeout"}

	// server_api
)

type Payload interface {
}

// notify async Call()
type RPCCallback_func func(Payload, RPCCallback_arg, error)
type RPCCallback_arg interface{}
