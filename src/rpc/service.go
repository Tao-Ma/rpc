// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

var (
	ErrServiceInvalidArg error = &Error{err: "service invalid argument"}
	ErrServiceNotInit    error = &Error{err: "service not init"}
)

type Service interface {
	// LoopOnce returns should continue.
	Loop(quit chan struct{})
	StopLoop(force bool)
	Cleanup()
}
