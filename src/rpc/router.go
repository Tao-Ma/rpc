// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"errors"
	"log"
	"net"
	"os"
	"time"
)

type EPWrapper interface {
	wrap(*EndPoint, Payload) Payload
}

type EndPoint struct {
	name string
	conn net.Conn

	r *Reader
	w *Writer

	in      chan Payload
	out     chan Payload
	wrapper EPWrapper

	hijacked bool // the endpoint is hijacked and read() is valid.

	logger *log.Logger
}

func NewEndPoint(name string, c net.Conn, out chan Payload, in chan Payload, wrapper EPWrapper, hf HeaderFactory, pf PayloadFactory, logger *log.Logger) *EndPoint {
	ep := new(EndPoint)

	ep.name = name
	ep.conn = c

	ep.in = in
	ep.out = out
	if ep.in != ep.out {
		// FIXME: not good ?
		ep.hijacked = true
	}

	ep.logger = logger

	ep.wrapper = wrapper

	ep.w = NewWriter(c, out, hf, pf, logger)
	ep.r = NewReader(c, in, ep, hf, pf, logger)

	return ep
}

func (ep *EndPoint) Run() {
	ep.r.Run()
	ep.w.Run()
}

func (ep *EndPoint) Stop() {
	ep.r.Stop()
	ep.w.Stop()
}

func (ep *EndPoint) Close() {
	ep.Stop()
	ep.conn.Close()
}

type msg struct {
	name string
	ep   *EndPoint
	p    Payload

	r      RpcMsg
	isresp bool
}

func (m *msg) GetPayloadId() uint16 {
	return m.p.GetPayloadId()
}

func (ep *EndPoint) wrap(p Payload) Payload {
	if ep.wrapper != nil {
		return ep.wrapper.wrap(ep, p)
	}
	return p
}

func (r *Router) wrap(ep *EndPoint, p Payload) Payload {
	m := new(msg)

	m.name = ep.name
	m.ep = ep
	m.p = p

	if rm, ok := p.(RpcMsg); ok {
		m.r = rm
		m.isresp = rm.RpcIsResponse()
	}

	return m
}

func (ep *EndPoint) read() (Payload, error) {
	if ep.hijacked {
		return ep.r.read()
	} else {
		// TODO: ERROR
		return nil, nil
	}
}

func (ep *EndPoint) Read() {
	if ep.hijacked {
		// TODO:
		return
	} else {
		panic("someone try to Read() a hijacked EndPoint")
		return
	}
}

func (ep *EndPoint) write(p Payload) error {
	return ep.w.Write(p)
}

type Listener struct {
	name string
	l    net.Listener

	hf HeaderFactory
	pf PayloadFactory

	r     *Router
	serve ServeConn

	bg *BackgroudService
}

func NewListener(name string, listener net.Listener, hf HeaderFactory, pf PayloadFactory, r *Router, serve ServeConn) (*Listener, error) {
	l := new(Listener)

	if bg, err := NewBackgroundService(l); err != nil {
		return nil, err
	} else {
		l.bg = bg
	}

	l.name = name
	l.l = listener
	l.hf = hf
	l.pf = pf

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

func (l *Listener) Close() {
	l.Stop()
	l.l.Close()
}

func (l *Listener) ServiceLoop(q chan bool, r chan bool) {
	go l.accepter(r)

	select {
	case <-q:
		l.Close()
	}
	// TODO: wait ?
}

func (l *Listener) accepter(r chan bool) {
	r <- true

	for {
		if c, err := l.l.Accept(); err != nil {
			// TODO: log?
			break
		} else if l.serve != nil && l.serve(l.r, c) {
			// TODO: serve
		} else if ep := l.r.newRouterEndPoint(l.name, c, l.hf, l.pf); ep == nil {
			// TODO: name!
			break
		} else {
			l.r.ep_in <- ep
		}
	}
}

type RpcMsg interface {
	RpcGetId() uint64
	RpcSetId(uint64)
	RpcIsRequest() bool
	RpcIsResponse() bool
}

type rpc_callback func(Payload, rpc_arg, error)
type rpc_arg interface{}

type rpc struct {
	id uint64

	name string
	ep   *EndPoint

	p     Payload
	r     RpcMsg
	isreq bool

	cb  rpc_callback
	arg rpc_arg
}

type Router struct {
	bg *BackgroudService

	nmap map[string]*EndPoint // used to find passive server
	lmap map[string]*Listener // Service name

	l_in   chan *Listener
	l_out  chan string
	ep_in  chan *EndPoint
	ep_out chan string

	out chan *rpc
	in  chan Payload

	next  uint64
	calls map[uint64]*rpc

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

	r.l_in = make(chan *Listener, 4)
	r.l_out = make(chan string, 16)
	r.ep_in = make(chan *EndPoint, 32)
	r.ep_out = make(chan string, 128)

	r.in = make(chan Payload, 1024*5)
	r.out = make(chan *rpc, 1024*5)

	r.calls = make(map[uint64]*rpc)
	r.next = 1

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
	r.bg.Stop()

	for _, l := range r.lmap {
		l.Close()
	}

	for _, ep := range r.nmap {
		ep.Close()
	}
}

func channelTimeoutWait(ch chan interface{}, v interface{}, timeout time.Duration) {
	select {
	case ch <- v:
	default:
		select {
		case ch <- v:
		case <-time.Tick(timeout):
		}

	}
}

// calldone is a helper to notify sync CallWait()
func calldone(p Payload, ch rpc_arg, err error) {
	// TODO: timeoud case may crash? Take care of the race condition!
	if ch, ok := ch.(chan Payload); !ok {
		panic("calldone")
	} else if err != nil {
		// TODO: error ?
		ch <- nil
	} else {
		ch <- p
	}
}

// Call sync
func (r *Router) CallWait(name string, p Payload, n time.Duration) Payload {
	ch := make(chan Payload, 1)

	r.Call(name, p, calldone, ch)

	// Wait result
	select {
	case p := <-ch:
		return p
	default:
		select {
		case p := <-ch:
			return p
		case <-time.Tick(n * time.Second):
			return nil
		}
	}

	// can not reach
	return nil
}

// Call async
func (r *Router) Call(name string, p Payload, cb rpc_callback, arg rpc_arg) {
	c := new(rpc)

	c.name = name
	c.cb = cb
	c.arg = arg
	c.p = p
	if m, ok := c.p.(RpcMsg); ok {
		c.r = m
		c.isreq = m.RpcIsRequest()
	}

	select {
	case r.out <- c:
	default:
		select {
		case r.out <- c:
		case <-time.Tick(0):
			// TODO: timeout!
			go cb(nil, arg, nil)
		}
	}
}

func (r *Router) Write(name string, p Payload) {
	r.Call(name, p, nil, nil)
}

// call internal
func (r *Router) write(ep *EndPoint, p Payload) {
	_ = ep.write(p)
}

// route()/hijack()
type ServeConn func(*Router, net.Conn) bool
type ServePayload func(*Router, string, Payload) bool

type RouteOut func(*Router, Payload)
type RouteIn func(*Router, Payload)

func (r *Router) newRouterEndPoint(name string, c net.Conn, hf HeaderFactory, pf PayloadFactory) *EndPoint {
	return NewEndPoint(name, c, make(chan Payload, 1024), r.in, r, hf, pf, r.logger)
}

func (r *Router) newHijackedEndPoint(name string, c net.Conn, hf HeaderFactory, pf PayloadFactory, logger *log.Logger) *EndPoint {
	return nil
}

func (r *Router) AddEndPoint(ep *EndPoint) error {
	select {
	case r.ep_in <- ep:
	default:
		select {
		case r.ep_in <- ep:
		case <-time.Tick(r.timeout):
			// TODO: track failure
		}
	}

	return nil
}

func (r *Router) DelEndPoint(name string) error {
	select {
	case r.ep_out <- name:
	default:
		select {
		case r.ep_out <- name:
		case <-time.Tick(r.timeout):
			// TODO: track failure
		}
	}

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
	select {
	case r.l_in <- l:
	default:
		select {
		case r.l_in <- l:
		case <-time.Tick(r.timeout):
			// TODO: error
		}
	}

	return nil
}

func (r *Router) DelListener(name string) error {
	select {
	case r.l_out <- name:
	default:
		select {
		case r.l_out <- name:
		case <-time.Tick(r.timeout):
			// TODO: track failure
		}
	}

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

func (r *Router) Dial(name string, network string, address string, hf HeaderFactory, pf PayloadFactory) error {
	if c, err := net.Dial(network, address); err != nil {
		return err
	} else if ep := r.newRouterEndPoint(name, c, hf, pf); ep == nil {
		c.Close()
		return err
	} else if err := r.AddEndPoint(ep); err != nil {
		// TODO: ep.Close()
		return err
	}

	return nil
}

func (r *Router) ListenAndServe(name string, network string, address string, hf HeaderFactory, pf PayloadFactory, server ServeConn) error {
	if l, err := net.Listen(network, address); err != nil {
		return err
	} else if l, err := NewListener(name, l, hf, pf, r, server); err != nil {
		l.Close()
		return err
	} else if err := r.AddListener(l); err != nil {
		l.Close()
		return err
	}

	return nil
}

func (r *Router) ServiceLoop(quit chan bool, ready chan bool) {
	ready <- true

forever:
	for {
		select {
		case <-quit:
			break forever
		case ep := <-r.ep_in:
			//r.logger.Printf("router: %v ep_in: %T:%v", r, ep, ep.name)
			r.addEndPoint(ep)
			ep.Run()
		case n := <-r.ep_out:
			if ep := r.delEndPoint(n); ep != nil {
				ep.Close()
			}
		case l := <-r.l_in:
			r.addListener(l)
			l.Run()
		case n := <-r.l_out:
			if l := r.delListener(n); l != nil {
				l.Close()
			}
		case c := <-r.out:
			//r.logger.Printf("router: %v send: %T:%v", r, c.p, c.p)
			// new payload read from api
			r.RpcOut(c)
			// TODO: route
			if ep, exist := r.nmap[c.name]; exist {
				//r.logger.Printf("router: %v rpcout: %T:%v", r, c.p, c.p)
				go r.write(ep, c.p)
			} else {
				// race condition: Dial() is later than Call()
				go c.cb(nil, c.arg, nil)
			}
		case p := <-r.in:
			//r.logger.Printf("router: %v recv: %T:%v", r, p, p)
			if m, ok := p.(*msg); !ok {
			} else {
				// TODO: route
				// new payload read from endpoint
				if c, err := r.RpcIn(m); err != nil {
					// TODO: not a rpc or rpc timeout/cancel
					go r.serve(r, m.name, m.p)
				} else {
					go c.cb(m.p, c.arg, nil)
				}
			}
		}
	}
}

func (r *Router) RpcOut(c *rpc) {
	if c.r != nil && c.isreq {
		c.id = r.next
		r.next++
		if _, exist := r.calls[c.id]; exist {
			panic("RpcOut id duplicate")
		} else {
			r.calls[c.id] = c
			c.r.RpcSetId(c.id)
		}
	}
}

// TODO: Payload will change
func (r *Router) RpcIn(m *msg) (*rpc, error) {
	if m.r != nil && m.isresp {
		id := m.r.RpcGetId()
		if c, exist := r.calls[id]; !exist {
			// timeout or cancel
			return nil, errors.New("")
		} else {
			return c, nil
		}
	}
	return nil, errors.New("")
}

// Endpoint.Read/Write
// route rule
// Error
// EndPoint/Listener must send join/left msg to sync with others!
// Log
// Performance
