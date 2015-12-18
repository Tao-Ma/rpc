// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"log"
	"net"
	"os"
	"time"
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

func (r *Router) Wrap(p Payload) RoutePayload {
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
		} else if ep := l.r.newRouterEndPoint(l.name, c, l.mf); ep == nil {
			// TODO: name!
			break
		} else {
			l.r.AddEndPoint(ep)
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
	go rm.cb(nil, rm.arg, nil)
}

// mock
func serve_done(p Payload, arg callback_arg, err error) {
	// Reclaim
}

func (rm *routeMsg) Serve(r *Router) {
	// TODO: Get the serve info
	// rm.rpc is used to locate method

	reply := r.serve(r, rm.ep_name, rm.p)

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
		select {
		case r.out <- out:
		case <-time.After(5):
			// TODO:
		}
	}
}

func (rm *routeMsg) Return(r *Router, reply RouteRPCPayload) {
	go rm.cb(reply.GetPayload(), rm.arg, nil)
	r.clientOutMsgs.Put(rm)
}

const (
	RouterOPAddEndPoint = iota
	RouterOPDelEndPoint
	RouterOPAddListener
	RouterOPDelListener
	RouterOPStop
)

type op struct {
	t int
	n string
	v interface{}
}

func (op *op) Reset() *op {
	op.n = ""
	op.v = nil
	return op
}

type Router struct {
	bg *BackgroudService

	nmap map[string]*EndPoint // used to find passive server
	lmap map[string]*Listener // Service name

	ops *ResourceManager
	op  chan *op

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

	r.lmap = make(map[string]*Listener)
	r.nmap = make(map[string]*EndPoint)

	op_num := 128
	r.ops = NewResourceManager(op_num, func() interface{} { return new(op) })
	r.op = make(chan *op, op_num)

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
	// Stop Listeners: No new EndPoint can be created.
	for _, l := range r.lmap {
		l.Stop()
	}

	// Stop Operations: No new EndPoint can be added.
	// TODO: in progress RPC?
	// Stop EndPoints: Server Shutdown/ClientShutdown.
	for _, ep := range r.nmap {
		ep.Stop()
	}

	// Reclaim resources
	r.waiters.Close()
	r.clientOutMsgs.Close()
	r.inMsgs.Close()
	r.serverOutMsgs.Close()

	// Stop Loop
	r.bg.Stop()

	// Close channels
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
func (r *Router) CallWait(ep string, rpc string, p Payload, n time.Duration) Payload {
	w := r.waiters.Get().(*waiter)

	// pass timeout information to Call.
	r.Call(ep, rpc, p, call_done, w)

	// wait result, rpc must returns something.
	result := <-w.ch

	r.waiters.Put(w)

	return result
}

// Call async
func (r *Router) Call(ep string, rpc string, p Payload, cb callback_func, arg callback_arg) {
	out := r.clientOutMsgs.Get().(*routeMsg)

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

	select {
	case r.out <- out:
	default:
		select {
		case r.out <- out:
		case <-time.Tick(0):
			// TODO: timeout!
			// TODO: RouteRPCPayload
			go cb(nil, arg, nil)
		}
	}
}

func (r *Router) Write(ep string, p Payload) {
	r.Call(ep, "", p, nil, nil)
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
	op := r.ops.Get().(*op)

	op.t = RouterOPAddEndPoint
	op.v = ep

	r.op <- op

	return nil
}

func (r *Router) DelEndPoint(name string) error {
	op := r.ops.Get().(*op)

	op.t = RouterOPDelEndPoint
	op.n = name

	r.op <- op

	return nil
}

// For client/accepter
func (r *Router) addEndPoint(ep *EndPoint) {
	if _, exist := r.nmap[ep.name]; !exist {
		r.nmap[ep.name] = ep
	} else {
		// oops! come here again? It is possible if reconnect and name does
		// not change.
	}

	return
}

// TODO: useless function?
func (r *Router) delEndPoint(name string) *EndPoint {
	if ep, exist := r.nmap[name]; exist {
		delete(r.nmap, name)
		return ep
	} else {
		return nil
	}
}

// For server
func (r *Router) AddListener(l *Listener) error {
	op := r.ops.Get().(*op)

	op.t = RouterOPAddListener
	op.v = l

	r.op <- op

	return nil
}

func (r *Router) DelListener(name string) error {
	op := r.ops.Get().(*op)

	op.t = RouterOPDelListener
	op.n = name

	r.op <- op

	return nil
}

func (r *Router) addListener(l *Listener) {
	if _, exist := r.lmap[l.name]; !exist {
		r.lmap[l.name] = l
	} else {
		// oops! come here again?
	}

	return
}

func (r *Router) delListener(name string) *Listener {
	if l, exist := r.lmap[name]; exist {
		delete(r.lmap, name)
		return l
	} else {
		return nil
	}
}

func (r *Router) Dial(name string, network string, address string, mf MsgFactory) error {
	if c, err := net.Dial(network, address); err != nil {
		return err
	} else if ep := r.newRouterEndPoint(name, c, mf); ep == nil {
		c.Close()
		return err
	} else if err := r.AddEndPoint(ep); err != nil {
		// TODO: ep.Stop()
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

func (r *Router) Loop(quit chan struct{}) {
forever:
	for {
		select {
		case <-quit:
			break forever
		case op := <-r.op:
			switch op.t {
			case RouterOPAddEndPoint:
				ep := op.v.(*EndPoint)
				r.addEndPoint(ep)
				ep.Run()
			case RouterOPDelEndPoint:
				if ep := r.delEndPoint(op.n); ep != nil {
					ep.Stop()
				}
			case RouterOPAddListener:
				l := op.v.(*Listener)
				r.addListener(l)
				l.Run()
			case RouterOPDelListener:
				if l := r.delListener(op.n); l != nil {
					l.Stop()
				}
			}
			r.ops.Put(op.Reset())
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
				go out.Error(nil)
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
		return out
	}
}

// route rule
// Error
// EndPoint/Listener must send join/left msg to sync with others!
