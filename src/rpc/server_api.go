// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"net"
)

// route()/hijack()
type ServeConn func(*Router, net.Conn) bool

// FIXME: Payload -> bool
type ServePayload func(*Router, string, Payload) Payload

// mock
func serve_done(p Payload, arg RPCCallback_arg, err error) {
	// Reclaim
}

func (rm *routeMsg) Serve(r *Router, ep_name string, rpc string, id uint64, p Payload) {
	// TODO: Get the serve info
	// rm.rpc is used to locate method

	reply := r.serve(r, ep_name, p)

	// TODO: nil?
	out := r.serverOutMsgs.Get().(*routeMsg)

	out.ep_name = ep_name
	out.rpc = rpc
	out.id = id

	out.p = reply

	out.is_rpc = true
	out.is_request = false

	out.cb = serve_done
	out.r = r

	select {
	case r.out <- out:
	default:
		panic("routeMsg leaks?!")
	}
}

func (rm *routeMsg) Return(r *Router, reply RouteRPCPayload) {
	go rm.cb(reply.GetPayload(), rm.arg, nil)
	rm.Recycle()
}
