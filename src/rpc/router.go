// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"log"
	"net"
	"os"
	"time"
)

var (
	ErrOPAddListenerStopping error = &Error{err: "AddListener is stopping"}
	ErrOPAddEndPointStopping error = &Error{err: "AddEndPoint is stopping"}
	ErrOPAddListenerExist    error = &Error{err: "AddEndPoint already exist"}
	ErrOPAddEndPointExist    error = &Error{err: "AddEndPoint already exist"}
	ErrOPListenerNotExist    error = &Error{err: "Listener does not exist"}
	ErrOPEndPointNotExist    error = &Error{err: "EndPoint does not exist"}

	ErrOutErrorEndPointNotExist error = &Error{err: "EndPoint does not exist"}
	ErrOPRouterStopped          error = &Error{err: "Router stopped"}

	ErrCallTimeout error = &Error{err: "Call timeout"}
)

type RPCInfo interface {
	GetRPCID() uint64
	SetRPCID(uint64)
	GetRPCName() string
	SetRPCName(string)

	IsRequest() bool
	SetIsRequest()
	IsReply() bool
	SetIsReply()
}

type Payload interface {
	// Raw
}

type RouteRPCPayload interface {
	RoutePayload
	RPCInfo

	// Run inside router goroutine
	Serve(*Router)
	Return(*Router, RouteRPCPayload)
}

type RoutePayload interface {
	IsRPC() bool
	SetIsRPC()

	GetEPName() string
	SetEPName(string)

	GetPayload() Payload
	Error(error)
	// AddStateChange api
}

type PayloadWrapper interface {
	Wrap(Payload) RoutePayload
	Unwrap(RoutePayload) Payload

	Error(*EndPoint, error)
}

type EndPoint struct {
	name string
	conn net.Conn

	r *Reader
	w *Writer

	pw PayloadWrapper

	in  chan Payload
	out chan Payload

	logger *log.Logger
}

func NewEndPoint(name string, c net.Conn, out chan Payload, in chan Payload, mf MsgFactory, pw PayloadWrapper, logger *log.Logger) *EndPoint {
	ep := new(EndPoint)

	ep.name = name
	ep.conn = c

	ep.in = in
	ep.out = out

	ep.pw = pw

	ep.logger = logger

	ep.w = NewWriter(c, ep, mf.NewBuffer(), logger)
	ep.r = NewReader(c, ep, mf.NewBuffer(), logger)

	return ep
}

func (ep *EndPoint) Run() {
	ep.r.Run()
	ep.w.Run()
}

func (ep *EndPoint) Stop() {
	ep.conn.Close()
	ep.r.Stop()
	ep.w.Stop()
}

func (ep *EndPoint) In() chan Payload {
	return ep.in
}

func (ep *EndPoint) Out() chan Payload {
	return ep.out
}

func (ep *EndPoint) Wrap(p Payload) Payload {
	if ep.pw == nil {
		return p
	}

	rp := ep.pw.Wrap(p)
	rp.SetEPName(ep.name)
	return Payload(rp)
}

func (ep *EndPoint) Unwrap(p Payload) Payload {
	if ep.pw == nil {
		return p
	}

	if rp, ok := p.(RoutePayload); ok {
		return ep.pw.Unwrap(rp)
	}

	// XXX: should not happen?
	return p
}

func (ep *EndPoint) InError(err error) {
	ep.pw.Error(ep, err)
}

func (ep *EndPoint) OutError(err error) {
	ep.pw.Error(ep, err)
}

func (r *Router) Error(ep *EndPoint, err error) {
	r.DelEndPoint(ep.name)
}

func (r *Router) Wrap(p Payload) RoutePayload {
	// TODO: shutdown safe? nil?
	in := r.inMsgs.Get().(*routeMsg)

	in.Reset()
	in.p = p

	return in
}

func (r *Router) Unwrap(p RoutePayload) Payload {
	if m, ok := p.(*routeMsg); ok {
		if !m.IsRPC() || !m.IsRequest() {
			r.serverOutMsgs.Put(m)
		}
		return m.p
	}

	// XXX: error?
	return p
}

func (ep *EndPoint) write(p RoutePayload) error {
	return ep.w.Write(p.(Payload))
}

type Listener struct {
	name string
	l    net.Listener

	mf MsgFactory

	r     *Router
	serve ServeConn

	bg *BackgroudService
}

func NewListener(name string, listener net.Listener, mf MsgFactory, r *Router, serve ServeConn) (*Listener, error) {
	l := new(Listener)

	if bg, err := NewBackgroundService(l); err != nil {
		return nil, err
	} else {
		l.bg = bg
	}

	l.name = name
	l.l = listener
	l.mf = mf

	// TODO: change this to interface
	l.r = r
	l.serve = serve

	return l, nil
}

func (l *Listener) Run() {
	l.bg.Run()
}

func (l *Listener) Stop() {
	l.bg.Stop()
}

func (l *Listener) StopLoop(force bool) {
	l.l.Close()
}

func (l *Listener) Loop(q chan struct{}) {
	go l.accepter()

	select {
	case <-q:
		l.StopLoop(false)
	}
	// TODO: wait ?
}

func (l *Listener) accepter() {
	for {
		if c, err := l.l.Accept(); err != nil {
			// TODO: log?
			break
		} else if l.serve != nil && l.serve(l.r, c) {
			// TODO: serve
		} else if ep := l.r.newRouterEndPoint(l.name+c.RemoteAddr().String(), c, l.mf); ep == nil {
			c.Close()
			break
		} else if err := l.r.AddEndPoint(ep); err != nil {
			ep.Stop()
		} else {
			//l.r.logger.Print(ep)
		}
	}
}

// serve
type callback_func func(Payload, callback_arg, error)
type callback_arg interface{}

type waiter struct {
	ch chan Payload
	// where we belones to
	r *Router
}

type routeMsg struct {
	id uint64

	ep_name    string
	rpc        string
	is_rpc     bool
	is_request bool

	p Payload

	to time.Time // ttl

	cb  callback_func
	arg callback_arg
}

func (rm *routeMsg) Reset() *routeMsg {
	rm.id = 0
	rm.ep_name = ""
	rm.rpc = ""
	rm.is_rpc = false
	rm.is_request = false
	rm.p = nil
	rm.cb = nil
	rm.arg = nil

	return rm
}

func (rm *routeMsg) GetEPName() string {
	return rm.ep_name
}

func (rm *routeMsg) SetEPName(name string) {
	rm.ep_name = name
}

func (rm *routeMsg) GetPayload() Payload {
	return rm.p
}

func (rm *routeMsg) IsRPC() bool {
	return rm.is_rpc
}

func (rm *routeMsg) SetIsRPC() {
	rm.is_rpc = true
}

func (rm *routeMsg) IsRequest() bool {
	return rm.is_request
}

func (rm *routeMsg) SetIsRequest() {
	rm.is_request = true
}

func (rm *routeMsg) IsReply() bool {
	return !rm.is_request
}

func (rm *routeMsg) SetIsReply() {
	rm.is_request = false
}

func (rm *routeMsg) GetRPCID() uint64 {
	return rm.id
}

func (rm *routeMsg) SetRPCID(id uint64) {
	rm.id = id
}

func (rm *routeMsg) GetRPCName() string {
	return rm.rpc
}

func (rm *routeMsg) SetRPCName(rpc string) {
	rm.rpc = rpc
}

func (rm *routeMsg) Error(err error) {
	// TODO: router?
	rm.cb(nil, rm.arg, err)
}

// mock
func serve_done(p Payload, arg callback_arg, err error) {
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

const (
	RouterOPAddEndPoint = iota
	RouterOPDelEndPoint
	RouterOPStopAddEndPoint
	RouterOPStopEndPoint
	RouterOPAddListener
	RouterOPDelListener
	RouterOPStopAddListener
	RouterOPStopListener
)

type opReq struct {
	t int
	n string
	v interface{}

	// error or object
	ret chan interface{}
}

func (op *opReq) Reset() *opReq {
	op.n = ""
	op.v = nil

	return op
}

type Router struct {
	bg *BackgroudService
	r  bool

	// Resource used to ask operation.
	ops *ResourceManager
	op  chan *opReq

	// Resources
	ep_stop  bool
	nmap     map[string]*EndPoint // used to find passive server
	lis_stop bool
	lmap     map[string]*Listener // Service name

	// protect by clientOutMsgs, serverOutMsgs, inMsgs
	out chan Payload
	in  chan Payload

	clientOutMsgs *ResourceManager
	serverOutMsgs *ResourceManager
	inMsgs        *ResourceManager

	waiters *ResourceManager

	next  uint64
	calls map[uint64]RouteRPCPayload

	serve ServePayload

	timeout time.Duration

	logger *log.Logger
}

func NewRouter(logger *log.Logger, serve ServePayload) (*Router, error) {
	r := new(Router)

	if bg, err := NewBackgroundService(r); err != nil {
		return nil, err
	} else {
		r.bg = bg
	}

	r.r = true

	r.lmap = make(map[string]*Listener)
	r.nmap = make(map[string]*EndPoint)

	op_num := 128
	r.op = make(chan *opReq, op_num)
	r.ops = NewResourceManager(op_num, func() interface{} { op := new(opReq); op.ret = make(chan interface{}, 1); return op })

	n := 1024
	r.in = make(chan Payload, n)
	r.out = make(chan Payload, n)

	r.waiters = NewResourceManager(n, func() interface{} { w := new(waiter); w.ch = make(chan Payload, 1); w.r = r; return w })
	r.calls = make(map[uint64]RouteRPCPayload)
	r.next = 1

	r.clientOutMsgs = NewResourceManager(n, func() interface{} { return new(routeMsg) })
	r.serverOutMsgs = NewResourceManager(n, func() interface{} { return new(routeMsg) })
	r.inMsgs = NewResourceManager(n, func() interface{} { return new(routeMsg) })

	r.serve = serve

	if logger == nil {
		r.logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		r.logger = logger
	}

	return r, nil
}

func (r *Router) Run() {
	r.bg.Run()
}

func (r *Router) Stop() {
	if !r.r {
		return
	}
	r.r = false

	// S:Stop Listeners: No new connection can be accepted.
	var op *opReq
	op = r.ops.Get().(*opReq)
	op.t = RouterOPStopAddListener
	r.op <- op
	if v := <-op.ret; v != nil {
		panic("stop add listener returns nil")
	}

stopListener:
	for {
		op = r.ops.Get().(*opReq)
		op.t = RouterOPStopListener
		r.op <- op
		v := <-op.ret
		switch t := v.(type) {
		case error:
			if t == ErrOPListenerNotExist {
				break stopListener
			} else {
				panic("")
			}
		case *Listener:
			t.Stop()
		default:
			panic("stop listener returns nil")
		}
	}

	// S/C:Stop EndPoints: Server Shutdown/Client Shutdown.
	op = r.ops.Get().(*opReq)
	op.t = RouterOPStopAddEndPoint
	r.op <- op
	if v := <-op.ret; v != nil {
		panic("stop add endpoint returns nil")
	}

stopEndPoint:
	for {
		op = r.ops.Get().(*opReq)
		op.t = RouterOPStopEndPoint
		r.op <- op
		v := <-op.ret
		switch t := v.(type) {
		case error:
			if t == ErrOPEndPointNotExist {
				break stopEndPoint
			} else {
				panic("")
			}
		case *EndPoint:
			t.Stop()
		default:
			panic("stop endpoint returns nil")
		}
	}

	// S/C:Stop Operations: No new EndPoint can be added.
	r.ops.Close()

	// C:TODO: in progress RPC?

	// Reclaim resources
	r.waiters.Close()
	r.clientOutMsgs.Close()
	r.inMsgs.Close()
	r.serverOutMsgs.Close()

	// Stop Loop
	r.bg.Stop()

	// Close channels
	close(r.op)
	close(r.in)
	close(r.out)
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

func (r *Router) call(ep string, rpc string, p Payload, cb callback_func, arg callback_arg, to time.Time) {
	var out *routeMsg
	if v := r.clientOutMsgs.Get(); v == nil {
		cb(nil, arg, ErrOPRouterStopped)
		return
	} else {
		out = v.(*routeMsg)
	}

	out.ep_name = ep
	out.rpc = rpc

	// Locate service name
	if rpc != "" {
		out.is_rpc = true
		out.is_request = true
	}

	out.p = p

	out.cb = cb
	out.arg = arg
	out.to = to

	select {
	case r.out <- out:
	default:
		panic("routeMsg leaks?!")
	}
}

func (r *Router) Write(ep string, p Payload) {
	r.Call(ep, "", p, nil, nil, 0)
}

// route()/hijack()
type ServeConn func(*Router, net.Conn) bool

// FIXME: Payload -> bool
type ServePayload func(*Router, string, Payload) Payload

func (r *Router) newRouterEndPoint(name string, c net.Conn, mf MsgFactory) *EndPoint {
	return NewEndPoint(name, c, make(chan Payload, 1024), r.in, mf, r, r.logger)
}

func (r *Router) newHijackedEndPoint(name string, c net.Conn, mf MsgFactory, logger *log.Logger) *EndPoint {
	return nil
}

func (r *Router) AddEndPoint(ep *EndPoint) error {
	var op *opReq
	if v := r.ops.Get(); v == nil {
		return ErrOPRouterStopped
	} else {
		op = v.(*opReq)
	}

	op.t = RouterOPAddEndPoint
	op.v = ep

	r.op <- op
	v := <-op.ret
	switch t := v.(type) {
	case error:
		return t
	case nil:
		return nil
	default:
		panic("AddEndPoint receive unexpected value")
	}
}

func (r *Router) DelEndPoint(name string) error {
	var op *opReq
	if v := r.ops.Get(); v == nil {
		return ErrOPRouterStopped
	} else {
		op = v.(*opReq)
	}

	op.t = RouterOPDelEndPoint
	op.n = name

	r.op <- op

	v := <-op.ret
	switch t := v.(type) {
	case error:
		return t
	case *EndPoint:
		t.Stop()
		return nil
	default:
		panic("del endpoint returns nil")
	}
}

// For client/accepter
func (r *Router) addEndPoint(ep *EndPoint) error {
	if _, exist := r.nmap[ep.name]; !exist {
		r.nmap[ep.name] = ep

		return nil
	}

	return ErrOPAddEndPointExist
}

// TODO: useless function?
func (r *Router) delEndPoint(name string) (*EndPoint, error) {
	if ep, exist := r.nmap[name]; exist {
		delete(r.nmap, name)
		return ep, nil
	}
	return nil, ErrOPEndPointNotExist
}

// For server
func (r *Router) AddListener(l *Listener) error {
	var op *opReq
	if v := r.ops.Get(); v == nil {
		return ErrOPRouterStopped
	} else {
		op = v.(*opReq)
	}

	l.Run()

	op.t = RouterOPAddListener
	op.v = l

	r.op <- op
	v := <-op.ret
	switch t := v.(type) {
	case error:
		return t
	case nil:
		return nil
	default:
		panic("AddListener receive unexpected value")
	}
}

func (r *Router) DelListener(name string) error {
	var op *opReq
	if v := r.ops.Get(); v == nil {
		return ErrOPRouterStopped
	} else {
		op = v.(*opReq)
	}

	op.t = RouterOPDelListener
	op.n = name

	r.op <- op
	v := <-op.ret
	switch t := v.(type) {
	case error:
		return t
	case *Listener:
		t.Stop()
		return nil
	default:
		panic("del listeners failed")
	}
}

func (r *Router) addListener(l *Listener) error {
	if _, exist := r.lmap[l.name]; !exist {
		r.lmap[l.name] = l
		return nil
	}

	return ErrOPAddListenerExist
}

func (r *Router) delListener(name string) (*Listener, error) {
	if l, exist := r.lmap[name]; exist {
		delete(r.lmap, name)
		return l, nil
	}

	return nil, ErrOPListenerNotExist
}

func (r *Router) Dial(name string, network string, address string, mf MsgFactory) error {
	if c, err := net.Dial(network, address); err != nil {
		return err
	} else if ep := r.newRouterEndPoint(name, c, mf); ep == nil {
		c.Close()
		return err
	} else if err := r.AddEndPoint(ep); err != nil {
		ep.Stop()
		return err
	}

	return nil
}

func (r *Router) ListenAndServe(name string, network string, address string, mf MsgFactory, server ServeConn) error {
	if l, err := net.Listen(network, address); err != nil {
		return err
	} else if l, err := NewListener(name, l, mf, r, server); err != nil {
		l.Stop()
		return err
	} else if err := r.AddListener(l); err != nil {
		l.Stop()
		return err
	}

	return nil
}

func (r *Router) StopLoop(force bool) {
}

func (r *Router) LoopProcessOperation(op *opReq) {
	var ret interface{}

	switch op.t {
	case RouterOPAddEndPoint:
		ep := op.v.(*EndPoint)
		if r.ep_stop {
			ret = ErrOPAddEndPointStopping
		} else {
			ret = r.addEndPoint(ep)
			ep.Run()
		}
	case RouterOPDelEndPoint:
		if ep, err := r.delEndPoint(op.n); err != nil {
			ret = err
		} else {
			ret = ep
		}
	case RouterOPAddListener:
		l := op.v.(*Listener)
		if r.lis_stop {
			ret = ErrOPAddListenerStopping
		} else {
			ret = r.addListener(l)
		}
	case RouterOPDelListener:
		if l, err := r.delListener(op.n); err != nil {
			ret = err
		} else {
			ret = l
		}

	case RouterOPStopAddListener:
		r.lis_stop = true
	case RouterOPStopAddEndPoint:
		r.ep_stop = true

	case RouterOPStopListener:
		ret = ErrOPListenerNotExist
		for k := range r.lmap {
			ret, _ = r.delListener(k)
			break
		}
	case RouterOPStopEndPoint:
		ret = ErrOPEndPointNotExist
		for k := range r.nmap {
			ret, _ = r.delEndPoint(k)
			break
		}
	}

	op.ret <- ret
	r.ops.Put(op.Reset())
}

func (r *Router) Loop(quit chan struct{}) {
forever:
	for {
		select {
		case <-quit:
			break forever
		case op := <-r.op:
			r.LoopProcessOperation(op)
		case p := <-r.out:
			//r.logger.Printf("router: %v send: %T:%v", r, c.p, c.p)
			out := p.(RoutePayload)

			if out.IsRPC() {
				r.RpcOut(out.(RouteRPCPayload))
			}

			// TODO: apply route rule

			if ep, exist := r.nmap[out.GetEPName()]; exist {
				//r.logger.Printf("router: %v rpcout: %T:%v", r, c.p, c.p)
				ep.write(out)
			} else {
				// race condition: Dial() is later than Call()
				go out.Error(ErrOutErrorEndPointNotExist)
			}
		case p := <-r.in:
			//r.logger.Printf("router: %v recv: %T:%v", r, p, p)
			in := p.(RoutePayload)

			// TODO: apply route rule

			if in.IsRPC() {
				if in.(RouteRPCPayload).IsRequest() {
					// rpc request
					in.(RouteRPCPayload).Serve(r)
				} else if out := r.RpcIn(in.(RouteRPCPayload)); out != nil {
					// rpc reply
					out.Return(r, in.(RouteRPCPayload))
				} else {
					// TODO: rpc timeout/cancel
				}
			} else {
				// TODO: msg
				go r.serve(r, in.GetEPName(), in.GetPayload())
			}
			r.inMsgs.Put(in)
		}
	}
}

func (r *Router) RpcOut(out RouteRPCPayload) {
	if !out.IsRequest() {
		return
	}

	out.SetRPCID(r.next)
	r.next++
	if _, exist := r.calls[out.GetRPCID()]; exist {
		panic("RpcOut id duplicate")
	} else {
		r.calls[out.GetRPCID()] = out
	}
}

func (r *Router) RpcIn(in RouteRPCPayload) RouteRPCPayload {
	if !in.IsReply() {
		return nil
	}

	id := in.GetRPCID()
	if out, exist := r.calls[id]; !exist {
		// timeout or cancel, the callback should be called.
		return nil
	} else {
		delete(r.calls, id)
		return out
	}
}
