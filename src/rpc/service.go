package rpc

import (
	"net"
	"time"
)

type EndPoint struct {
	conn net.Conn

	r *Reader
	w *Writer

	d      Delegator
	router Router
}

func newEndPoint(conn net.Conn, hf HeaderFactory, pf PayloadFactory, r Router) (*EndPoint, error) {

	ep := new(EndPoint)

	if conn == nil || pf == nil || hf == nil || r == nil {
		panic("newEndPoint init failed")
	}

	ep.conn = conn
	ep.w = NewWriter(conn, hf, pf)
	ep.r = NewReader(conn, hf, pf, r)

	return ep, nil
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

	if ep.d != nil {
		ep.d.Free(ep)
	}
}

type Accepter struct {
	bg *BackgroudService
	s  ServeService
	l  net.Listener
	rm *ResourceManager
}

func NewAccepter(s ServeService, l net.Listener) (*Accepter, error) {
	a := new(Accepter)

	if bg, err := NewBackgroundService(a); err != nil {
		return nil, err
	} else {
		a.bg = bg
	}

	a.s = s
	a.l = l

	a.rm, _ = NewResourceManager()

	return a, nil
}

func (a *Accepter) Run() {
	a.bg.Run()
}

func (a *Accepter) Stop() {
	a.bg.Stop()
}

func (a *Accepter) ServiceLoop(q chan bool, r chan bool) {
	go a.accepter(a.l, r)

	select {
	case <-q:
		_ = a.l.Close()
	}

	// TODO: wait accepter?
}

func (a *Accepter) accepter(l net.Listener, r chan bool) {
	hf := a.s.GetHeaderFactory()
	pf := a.s.GetPayloadFactory()
	router := a.s.GetRouter()

	r <- true
	for {
		if c, err := l.Accept(); err != nil {
			// TODO: log?
			return
		} else if ep, err := newEndPoint(c, hf, pf, router); err != nil {
			// TODO: log!
		} else {
			a.rm.Pass(ep)
		}
	}
}

type Resource interface {
	Run()
	Stop()
}

type Delegator interface {
	Pass(interface{})
	Free(interface{})
}

type ResourceManager struct {
	bg *BackgroudService

	n    uint32
	pass chan Resource
	free chan Resource
	eps  map[Resource]Resource

	pass_timeout time.Duration
	free_timeout time.Duration
}

func NewResourceManager() (*ResourceManager, error) {
	m := new(ResourceManager)

	if bg, err := NewBackgroundService(m); err != nil {
		return nil, err
	} else {
		m.bg = bg
	}

	m.n = 10

	m.pass = make(chan Resource, m.n)
	m.free = make(chan Resource, m.n)
	m.eps = make(map[Resource]Resource)

	m.pass_timeout = 3000 * time.Millisecond
	m.free_timeout = 3000 * time.Millisecond

	return m, nil
}

func (m *ResourceManager) Run() {
	m.bg.Run()
}

func (m *ResourceManager) Stop() {
	m.bg.Stop()
}

func (m *ResourceManager) Pass(v interface{}) error {
	// TODO: Check running state ?
	var r Resource

	if v, ok := v.(Resource); !ok {
		return nil
	} else {
		r = v
	}

	select {
	case m.pass <- r:
	default:
		select {
		case m.pass <- r:
		case <-time.Tick(m.pass_timeout):
			return nil
		}
	}
	return nil
}

func (m *ResourceManager) Free(v interface{}) error {
	// TODO: Check running state ?
	var r Resource

	if v, ok := v.(*EndPoint); !ok {
		return nil
	} else {
		r = v
	}

	select {
	case m.free <- r:
	default:
		select {
		case m.free <- r:
		case <-time.Tick(m.free_timeout):
			return nil
		}
	}
	return nil
}

func (m *ResourceManager) ServiceLoop(q chan bool, r chan bool) {
	r <- true

forever:
	for {
		select {
		case <-q:
			break forever
		case r := <-m.pass:
			if _, exist := m.eps[r]; exist {
				m.eps[r] = r
				r.Run()
			}
		case r := <-m.free:
			if _, exist := m.eps[r]; exist {
				delete(m.eps, r)
				r.Stop()
			}
		}
	}
}

type Service interface {
	GetHeaderFactory() HeaderFactory
	GetPayloadFactory() PayloadFactory
	GetRouter() Router
}

type DialService interface {
	Service

	Dial(string, string) error
}

type ServeService interface {
	Service

	ListenAndServe(string, string) error
}

type DefaultService struct {
	hf HeaderFactory
	pf PayloadFactory

	r  Router
	t  *Tracker
	a  *Accepter
	rm *ResourceManager
}

func newService(hf HeaderFactory, pf PayloadFactory, r Router) (*DefaultService, error) {
	s := new(DefaultService)

	s.hf = hf
	s.pf = pf
	s.r = r

	s.t = NewRPCTracker(2048, s.r)
	s.rm, _ = NewResourceManager()

	return s, nil
}

func newDefaultService(r Router) (*DefaultService, error) {
	return newService(NewDefaultHeaderFactory(), NewProtobufFactory(), r)
}

func (s *DefaultService) GetRouter() Router {
	return s.r
}

func (s *DefaultService) GetHeaderFactory() HeaderFactory {
	return s.hf
}

func (s *DefaultService) GetPayloadFactory() PayloadFactory {
	return s.pf
}

func (s *DefaultService) Run() {
	s.t.Run()
	if s.a != nil {
		s.a.Run()
	}
}

func (s *DefaultService) Stop() {
	s.t.Stop()
	if s.a != nil {
		s.a.Stop()
	}
}

func NewDefaultServeService(r Router) (ServeService, error) {
	if ss, err := newDefaultService(r); err != nil {
		return nil, err
	} else {
		return ServeService(ss), nil
	}
}

func (s *DefaultService) ListenAndServe(network string, addr string) error {
	l, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	s.a, _ = NewAccepter(s, l)
	return nil
}

func NewDefaultDialService(r Router) (DialService, error) {
	if ds, err := newDefaultService(r); err != nil {
		return nil, err
	} else {
		return DialService(ds), nil
	}
}

func (s *DefaultService) Dial(network string, addr string) error {
	c, err := net.Dial(network, addr)
	if err != nil {
		return err
	}

	if ep, err := newEndPoint(c, s.hf, s.pf, s.r); err != nil {
		return err
	} else {
		s.rm.Pass(ep)
		return nil
	}
}
