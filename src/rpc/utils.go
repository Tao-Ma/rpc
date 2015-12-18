// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

type Error struct {
	err string
}

func (e *Error) Error() string {
	return e.err
}
