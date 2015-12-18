// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

var ()

type new_func func() interface{}

type ResourceManager struct {
	n int

	ch chan interface{}
}

func NewResourceManager(n int, f new_func) *ResourceManager {
	rm := new(ResourceManager)

	rm.n = n

	rm.ch = make(chan interface{}, n)

	for i := 0; i < n; i++ {
		rm.ch <- f()
	}

	return rm
}

func (rm *ResourceManager) Close() {
	for rm.n > 0 {
		if v := rm.Get(); v != nil {
			rm.n--
		}
	}

	close(rm.ch)
}

func (rm *ResourceManager) Get() interface{} {
	select {
	case v := <-rm.ch:
		return v
	}
}

func (rm *ResourceManager) Put(v interface{}) {
	select {
	case rm.ch <- v:
	}
}
