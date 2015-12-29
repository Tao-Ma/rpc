// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"time"
)

var (
	ErrCallTimeout error = &Error{err: "Call timeout"}
)

type waiter struct {
	ch chan Payload
	// where we belones to
	r *Router
}

// call_done is a helper to notify sync CallWait()
func call_done(p Payload, arg callback_arg, err error) {
	// TODO: timeout case may crash? Take care of the race condition!
	if w, ok := arg.(*waiter); !ok {
		panic("call_done")
	} else if err != nil {
		// TODO: error ?
		w.ch <- nil
	} else {
		w.ch <- p
	}
}

// Call sync
func (r *Router) CallWait(ep string, rpc string, p Payload, n time.Duration) (Payload, error) {
	if n < 0 {
		return nil, ErrCallTimeout
	} else if n == 0 {
		// long enough
		n = 5 * time.Minute
	} else {
		n = n * time.Second
	}

	to := time.Now().Add(n)

	var w *waiter
	if v := r.waiters.Get(); v == nil {
		return nil, ErrOPRouterStopped
	} else {
		w = v.(*waiter)
	}

	// pass timeout information to Call.
	r.call(ep, rpc, p, call_done, w, to)
	// wait result, rpc must returns something.
	result := <-w.ch

	r.waiters.Put(w)

	return result, nil
}

// Call async
func (r *Router) Call(ep string, rpc string, p Payload, cb callback_func, arg callback_arg, n time.Duration) {
	if n < 0 {
		cb(nil, arg, ErrCallTimeout)
		return
	} else if n == 0 {
		n = 5 * time.Minute
	} else {
		n = n * time.Second
	}

	r.call(ep, rpc, p, cb, arg, time.Now().Add(n))
}
