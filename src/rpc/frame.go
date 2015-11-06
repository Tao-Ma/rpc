// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import (
	"io"
	"time"
)

type Header interface {
	GetLen() uint32
	Marshal() ([]byte, error)
	Unmarshal([]byte) error

	SetPayloadId(uint16)
	GetPayloadId() uint16
	SetPayloadLen(uint32)
	GetPayloadLen() uint32
}

type Payload interface {
	GetPayloadId() uint16
	MarshalPayload() ([]byte, error)
	UnmarshalPayload([]byte) error
}

type HeaderFactory interface {
	New() Header
}

type PayloadFactory interface {
	New(id uint16) Payload
}

type ServiceLoop interface {
	ServiceLoop(chan bool, chan bool)
}

type BackgroudService struct {
	running bool
	err     error

	quit  chan bool
	ready chan bool

	l ServiceLoop

	start_timeout time.Duration
	stop_timeout  time.Duration
}

func NewBackgroundService(l ServiceLoop) (*BackgroudService, error) {
	bg := new(BackgroudService)

	bg.l = l
	bg.start_timeout = 3000 * time.Millisecond
	bg.stop_timeout = 3000 * time.Millisecond

	return bg, nil
}

func (bg *BackgroudService) Run() {
	if bg.running {
		return
	}

	bg.quit = make(chan bool, 1)
	bg.ready = make(chan bool, 1)

	go bg.l.ServiceLoop(bg.quit, bg.ready)

	select {
	case <-bg.ready:
		bg.running = true
	case <-time.Tick(bg.start_timeout):
		close(bg.quit)
		// TODO: bg.err
		break
	}

	close(bg.ready)
}

func (bg *BackgroudService) Stop() {
	if !bg.running {
		return
	}

	select {
	case bg.quit <- true:
	case <-time.Tick(bg.stop_timeout):
		// TODO: bg.err
	}

	close(bg.quit)
	bg.running = false
}

type Writer struct {
	bg *BackgroudService

	conn io.Writer
	hf   HeaderFactory
	pf   PayloadFactory
	err  error

	out chan *[]byte
}

func NewWriter(conn io.Writer, hf HeaderFactory, pf PayloadFactory) *Writer {
	w := new(Writer)

	if bg, err := NewBackgroundService(w); err != nil {
		return nil
	} else {
		w.bg = bg
	}

	w.conn = conn
	w.hf = hf
	w.pf = pf

	w.out = make(chan *[]byte)

	return w
}

func (w *Writer) Run() {
	w.bg.Run()
}

func (w *Writer) Stop() {
	w.bg.Stop()
}

func (w *Writer) Write(p Payload) error {
	var (
		pb []byte
		b  []byte
	)

	if b, err := p.MarshalPayload(); err != nil {
		return err
	} else {
		pb = b
	}

	hdr := w.hf.New()
	hdr.SetPayloadId(p.GetPayloadId())
	hdr.SetPayloadLen(uint32(len(pb)))
	if hb, err := hdr.Marshal(); err != nil {
		return err
	} else {
		b = append(hb, pb...)
		hb = nil
		pb = nil
	}

	select {
	case w.out <- &b:
	default:
		select {
		case w.out <- &b:
		case <-time.Tick(time.Second):
			// timeout case
			return nil
		}
	}

	// TODO: block support
	return nil
}

func (w *Writer) ServiceLoop(q chan bool, r chan bool) {
	r <- true
forever:
	for {
		// forward msg from chan to conn
		select {
		case <-q:
			break forever
		case b := <-w.out:
			// TODO: timeout or error?
			if _, err := w.conn.Write(*b); err != nil {
				w.err = err
				// TODO: stop the writer?
				break forever
			}
			break
		}
	}
}

type Reader struct {
	bg *BackgroudService

	conn   io.Reader
	hf     HeaderFactory
	pf     PayloadFactory
	err    error
	maxlen uint32

	r Router
}

func NewReader(conn io.Reader, hf HeaderFactory, pf PayloadFactory, router Router) *Reader {
	r := new(Reader)

	if bg, err := NewBackgroundService(r); err != nil {
		return nil
	} else {
		r.bg = bg
	}

	r.conn = conn
	r.hf = hf
	r.pf = pf
	r.maxlen = 4096

	r.r = router

	return r
}

func (r *Reader) Run() {
	r.bg.Run()
}

func (r *Reader) Stop() {
	r.bg.Stop()
}

func (r *Reader) ServiceLoop(q chan bool, ready chan bool) {
	ready <- true
forever:
	for {
		if err := r.read(); err != nil {
			// TODO: timeout or error?
			r.err = err
			break forever
		}

		select {
		case <-q:
			// TODO: cleanup
			break forever
		default:
		}
	}
}

//Try to read a whole msg.
func (r *Reader) read() error {
	hdr := r.hf.New()

	// TODO: Read() interrupt?
	b := make([]byte, hdr.GetLen(), r.maxlen+hdr.GetLen())
	if _, err := r.conn.Read(b); err != nil {
		b = nil
		return err
	}

	if err := hdr.Unmarshal(b); err != nil {
		b = nil
		return err
	}

	plen := hdr.GetPayloadLen()
	// Support header-only message(udp or no payload at all).
	if plen > 0 {
		// TODO: enlarge b []byte if plen > r.maxlen or error out.
		if n, err := r.conn.Read(b[hdr.GetLen() : hdr.GetLen()+plen]); err != nil {
			b = nil
			return err
		} else if uint32(n) != plen {
		}
	}

	p := r.pf.New(hdr.GetPayloadId())
	if err := p.UnmarshalPayload(b[hdr.GetLen() : hdr.GetLen()+plen]); err != nil {
		return err
	}

	switch v := p.(type) {
	case RpcRequest:
		r.r.RpcRequestRoute(v)
	case RpcResponse:
		r.r.RpcResponseRoute(v, nil)
	default:
		r.r.DefaultRoute(p)
	}

	return nil
}
