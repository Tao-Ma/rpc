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
)

type RPCInfo interface {
	GetRPCID() uint64
	SetRPCID(uint64)
	GetRPCName() string
	SetRPCName(string)

	GetTrackID() TrackID
	SetTrackID(TrackID)
	TrackObject

	IsRequest() bool
	SetIsRequest()
	IsReply() bool
	SetIsReply()
}

type RouteRPCPayload interface {
	RoutePayload
	RPCInfo

	// Run inside router goroutine
	Serve(*Router, string, string, uint64, Payload)
	Return(*Router, RouteRPCPayload)
}

type RoutePayload interface {
	Resource
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
	ep.Cleanup()
}

func (ep *EndPoint) Cleanup() {
	// TODO: use Router.CleanupEndPoint
cleanup:
	for {
		select {
		case p := <-ep.out:
			// TODO: generalize interface
			p.(*routeMsg).Recycle()
		default:
			break cleanup
		}
	}

	close(ep.out)
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
			m.Recycle()
		}
		return m.p
	}

	panic("Router.Unwrap invalid type")
	return p
}

func (ep *EndPoint) write(p RoutePayload) error {
	return ep.w.Write(p.(Payload))
}

type routeMsg struct {
	id  uint64
	tid TrackID

	owner *ResourceManager

	ep_name    string
	rpc        string
	is_rpc     bool
	is_request bool

	p Payload

	r  *Router   // owner
	to time.Time // ttl

	cb  RPCCallback_func
	arg RPCCallback_arg
}

func (rm *routeMsg) Recycle() {
	rm.owner.Put(rm)
}

func (rm *routeMsg) SetOwner(o *ResourceManager) Resource {
	rm.owner = o
	return rm
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

func (rm *routeMsg) GetTrackID() TrackID {
	return rm.tid
}

func (rm *routeMsg) SetTrackID(tid TrackID) {
	rm.tid = tid
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

func (rm *routeMsg) When() time.Time {
	return rm.to
}

func (rm *routeMsg) Timeout(now time.Time) {
	rm.r.stats.rpcTimeout++
	go rm.cb(nil, rm.arg, ErrCallTimeout)
	r := rm.r
	if out, exist := r.calls[rm.GetRPCID()]; exist {
		delete(r.calls, rm.GetRPCID())
		// TODO: Redesign the api
		out.(*routeMsg).Recycle()
	}
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

type Chan struct {
	ch chan interface{}

	owner *ResourceManager
}

func NewChan() *Chan {
	ch := new(Chan)
	ch.ch = make(chan interface{}, 1)
	return ch
}

func (ch *Chan) SetOwner(o *ResourceManager) Resource {
	ch.owner = o
	return ch
}

func (ch *Chan) Recycle() {
	ch.owner.Put(ch)
}

func (ch *Chan) Reset() Resource {
	// do nothing
	return ch
}

type opReq struct {
	t int
	n string

	// ref
	v interface{}

	// error or object
	ret *Chan

	owner *ResourceManager
}

func (op *opReq) Recycle() {
	op.owner.Put(op)
}

func (op *opReq) SetOwner(o *ResourceManager) Resource {
	op.owner = o
	return op
}

func (op *opReq) Reset() *opReq {
	op.n = ""
	op.v = nil

	return op
}

type routerStats struct {
	epIn  uint64
	epOut uint64

	rpcIn      uint64
	rpcOut     uint64
	rpcTimeout uint64
	rpcError   uint64
}

type Router struct {
	bg *BackgroudService
	r  bool

	// Resource used to ask operation.
	ops   *ResourceManager // operation request
	opchs *ResourceManager // operation request notify channel
	op    chan *opReq

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

	tt *TimeoutTracker

	serve ServePayload

	timeout time.Duration

	stats routerStats

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

	op_num := 16
	r.op = make(chan *opReq, op_num)
	r.ops = NewResourceManager(op_num, func() Resource { op := new(opReq); return op })
	r.opchs = NewResourceManager(op_num, func() Resource { return NewChan() })

	n := 1 * 128
	r.in = make(chan Payload, n)
	r.out = make(chan Payload, n*2)

	r.waiters = NewResourceManager(n, func() Resource { w := new(waiter); w.ch = make(chan Payload, 1); w.r = r; return w })
	r.calls = make(map[uint64]RouteRPCPayload)
	r.next = 1
	r.tt, _ = NewTimeoutTracker(100, n)

	r.clientOutMsgs = NewResourceManager(n, func() Resource { return new(routeMsg) })
	r.serverOutMsgs = NewResourceManager(n, func() Resource { return new(routeMsg) })
	r.inMsgs = NewResourceManager(n, func() Resource { return new(routeMsg) })

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

func (r *Router) requestOP(t int, obj ...interface{}) (interface{}, error) {
	var v_ch interface{}
	var v_obj interface{}
	var v_n string

	v_ch = r.opchs.Get()
	if v_ch == nil {
		return nil, ErrOPRouterStopped
	}

	for _, v := range obj {
		switch t := v.(type) {
		case *Listener:
			v_obj = t
		case *EndPoint:
			v_obj = t
		case string:
			v_n = t
		default:
			panic("Router.requestOP recieve unexpected object")
		}
	}

	ch := v_ch.(*Chan)
	op := r.ops.Get().(*opReq)
	op.t = t
	op.v = v_obj
	op.n = v_n
	op.ret = ch
	r.op <- op
	v := <-ch.ch
	ch.Recycle()
	return v, nil
}

func (r *Router) Cleanup() {
	// nothing to do
}

func (r *Router) Stop() {
	if !r.r {
		return
	}
	r.r = false

	// S:Stop Listeners: No new connection can be accepted.
	if v, _ := r.requestOP(RouterOPStopAddListener); v != nil {
		panic("stop add listener returns nil")
	}

stopListener:
	for {
		v, _ := r.requestOP(RouterOPStopListener)
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
	if v, _ := r.requestOP(RouterOPStopAddEndPoint); v != nil {
		panic("stop add endpoint returns nil")
	}

stopEndPoint:
	for {
		v, _ := r.requestOP(RouterOPStopEndPoint)
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
	r.opchs.Close()
	r.ops.Close()

	// C:TODO: in progress RPC?

	r.tt.Stop()

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

	//	r.logger.Printf("rpc in: %v rpc out: %v rpc err: %v rpc timeout: %v\n",
	//		r.stats.rpcIn, r.stats.rpcOut, r.stats.rpcError, r.stats.rpcTimeout)
}

func (r *Router) call(ep string, rpc string, p Payload, cb RPCCallback_func, arg RPCCallback_arg, to time.Time) {
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

	out.r = r

	select {
	case r.out <- out:
	default:
		panic("routeMsg leaks?!")
	}
}

func (r *Router) Write(ep string, p Payload) {
	r.Call(ep, "", p, nil, nil, 0)
}

func (r *Router) newRouterEndPoint(name string, c net.Conn, mf MsgFactory) *EndPoint {
	return NewEndPoint(name, c, make(chan Payload, 8*128), r.in, mf, r, r.logger)
}

func (r *Router) newHijackedEndPoint(name string, c net.Conn, mf MsgFactory, logger *log.Logger) *EndPoint {
	return nil
}

func (r *Router) AddEndPoint(ep *EndPoint) error {
	v, err := r.requestOP(RouterOPAddEndPoint, ep)
	if err != nil {
		return err
	}

	switch t := v.(type) {
	case error:
		return t
	case nil:
		return nil
	default:
		r.logger.Print(t)
		panic("AddEndPoint receive unexpected value")
	}
}

func (r *Router) DelEndPoint(name string) error {
	v, err := r.requestOP(RouterOPDelEndPoint, name)
	if err != nil {
		return err
	}
	switch t := v.(type) {
	case error:
		return t
	case *EndPoint:
		// TODO: task queue
		go t.Stop()
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
	v, err := r.requestOP(RouterOPAddListener, l)
	if err != nil {
		return err
	}

	l.Run()

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
	v, err := r.requestOP(RouterOPDelListener, name)
	if err != nil {
		return err
	}
	switch t := v.(type) {
	case error:
		return t
	case *Listener:
		// TODO: task queue
		go t.Stop()
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

	ch := op.ret
	ch.ch <- ret
	op.Reset().Recycle()
}

func (r *Router) ProcessOut(out RoutePayload) {
	//r.logger.Printf("router: %v send: %T:%v", r, c.p, c.p)
	if out.IsRPC() {
		r.RpcOut(out.(RouteRPCPayload))
	}

	// TODO: apply route rule

	if ep, exist := r.nmap[out.GetEPName()]; exist {
		//r.logger.Printf("router: %v rpcout: %T:%v", r, c.p, c.p)
		if err := ep.write(out); err != nil {
			r.stats.rpcError++
			go out.Error(err)
			// TODO: redesign the api
			out.(*routeMsg).Recycle()
		}
	} else {
		// race condition: Dial() is later than Call()
		r.stats.rpcError++
		go out.Error(ErrOutErrorEndPointNotExist)
		// TODO: redesign the api
		out.(*routeMsg).Recycle()
	}
}

func (r *Router) ProcessIn(in RoutePayload) {
	//r.logger.Printf("router: %v recv: %T:%v", r, p, p)
	rm := in.(*routeMsg)

	// TODO: apply route rule

	if in.IsRPC() {
		if in.(RouteRPCPayload).IsRequest() {
			// rpc request
			// TODO: task queue
			// TODO: server api
			go in.(RouteRPCPayload).Serve(r, rm.ep_name, rm.rpc, rm.id, rm.p)
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
	// TODO: redesign the api
	in.(*routeMsg).Recycle()
}

func (r *Router) Loop(quit chan struct{}) {
forever:
	for {
		select {
		case <-quit:
			break forever
		case op := <-r.op:
			r.LoopProcessOperation(op)
		case now := <-r.tt.Tick():
			r.tt.TimeoutCheck(now)
		case p := <-r.out:
			r.ProcessOut(p.(RoutePayload))
		case p := <-r.in:
			r.ProcessIn(p.(RoutePayload))
		}
	}
}

func (r *Router) RpcOut(out RouteRPCPayload) {
	if !out.IsRequest() {
		return
	}

	r.stats.rpcOut++
	out.SetRPCID(r.next)
	r.next++
	if _, exist := r.calls[out.GetRPCID()]; exist {
		panic("RpcOut id duplicate")
	} else {
		r.calls[out.GetRPCID()] = out
		id, _ := r.tt.Add(out)
		out.SetTrackID(id)
	}
}

func (r *Router) RpcIn(in RouteRPCPayload) RouteRPCPayload {
	if !in.IsReply() {
		return nil
	}

	r.stats.rpcIn++
	id := in.GetRPCID()
	if out, exist := r.calls[id]; !exist {
		// timeout or cancel, the callback should be called.
		return nil
	} else {
		delete(r.calls, id)
		r.tt.Del(out.GetTrackID())
		return out
	}
}
