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

func (rm *routeMsg) Serve(r *Router) {
	// TODO: Get the serve info
	// rm.rpc is used to locate method

	reply := r.serve(r, rm.ep_name, rm.p)

	// TODO: nil?
	out := r.serverOutMsgs.Get().(*routeMsg)

	out.ep_name = rm.ep_name
	out.rpc = rm.rpc
	out.id = rm.id

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
	r.clientOutMsgs.Put(rm)
}
