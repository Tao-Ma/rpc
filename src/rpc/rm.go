// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

var ()

type Resource interface {
	// Reset()
	SetOwner(*ResourceManager) Resource
}

type new_func func() Resource

type ResourceManager struct {
	n int
	r bool

	ch chan Resource
}

func NewResourceManager(n int, f new_func) *ResourceManager {
	rm := new(ResourceManager)

	rm.n = n

	rm.ch = make(chan Resource, n)

	for i := 0; i < n; i++ {
		rm.ch <- f().SetOwner(rm)
	}

	rm.r = true

	return rm
}

func (rm *ResourceManager) Close() {
	if !rm.r {
		return
	}

	for rm.n > 0 {
		if r := rm.Get(); r != nil {
			rm.n--
		}
	}

	close(rm.ch)
	rm.r = false
}

func (rm *ResourceManager) Get() Resource {
	select {
	case r := <-rm.ch:
		return r
	}
}

func (rm *ResourceManager) Put(r Resource) {
	select {
	case rm.ch <- r:
	}
}
