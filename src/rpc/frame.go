// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"io"
	"log"
	"os"
	"time"
)

type Payload interface{}

type ReaderWrapper interface {
	wrap(Payload) Payload
}

type MsgFactory interface {
	NewBufferFactory() MsgBufferFactory
}

type MsgBufferFactory interface {
	MarshalHeader([]byte, Payload, uint32) error
	UnmarshalHeader([]byte) error

	MarshalPayload(Payload, []byte) ([]byte, error)
	UnmarshalPayload([]byte) (Payload, error)

	GetHdrLen() uint32
	GetPayloadLen() uint32
}

type ServiceLoop interface {
	ServiceLoop(chan struct{}, chan bool)
}

type BackgroudService struct {
	running bool
	err     error

	quit  chan struct{}
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

	bg.quit = make(chan struct{}, 1)
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

	bg.running = false
	close(bg.quit)
}

type Writer struct {
	bg *BackgroudService

	conn io.Writer
	mbf  MsgBufferFactory
	err  error

	out chan Payload

	maxlen uint32
	b      []byte

	logger *log.Logger
}

func NewWriter(conn io.Writer, out chan Payload, mf MsgFactory, logger *log.Logger) *Writer {
	w := new(Writer)

	if bg, err := NewBackgroundService(w); err != nil {
		return nil
	} else {
		w.bg = bg
	}

	w.conn = conn
	w.mbf = mf.NewBufferFactory()

	w.out = out

	w.maxlen = 4096
	w.b = make([]byte, w.mbf.GetHdrLen()+w.maxlen)

	if logger == nil {
		w.logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		w.logger = logger
	}

	return w
}

func (w *Writer) Run() {
	w.bg.Run()
}

func (w *Writer) Stop() {
	w.bg.Stop()
}

func (w *Writer) Write(p Payload) error {
	select {
	case w.out <- p:
	default:
		select {
		case w.out <- p:
		case <-time.Tick(0 * time.Second):
			// TODO: timeout
			return nil
		}
	}

	return nil
}

func (w *Writer) write(p Payload) ([]byte, error) {
	pb, err := w.mbf.MarshalPayload(p, w.b[w.mbf.GetHdrLen():w.mbf.GetHdrLen()])
	if err != nil {
		return nil, err
	}

	hb := w.b[0:w.mbf.GetHdrLen()]
	if err := w.mbf.MarshalHeader(hb, p, uint32(len(pb))); err != nil {
		return nil, err
	}

	b := append(hb, pb...)

	return b, nil
}

func (w *Writer) ServiceLoop(q chan struct{}, r chan bool) {
	r <- true
forever:
	for {
		// forward msg from chan to conn
		select {
		case <-q:
			break forever
		case p := <-w.out:
			// TODO: timeout or error?
			if b, err := w.write(p); err != nil {
				// TODO: error?
			} else if _, err := w.conn.Write(b); err != nil {
				w.err = err
				// TODO: stop the writer?
				break forever
			}
		}
	}
}

type Reader struct {
	bg *BackgroudService

	conn io.Reader
	mbf  MsgBufferFactory
	err  error

	in      chan Payload
	wrapper ReaderWrapper

	maxlen uint32
	b      []byte

	logger *log.Logger
}

func NewReader(conn io.Reader, in chan Payload, wrapper ReaderWrapper, mf MsgFactory, logger *log.Logger) *Reader {
	r := new(Reader)

	if wrapper == nil {
		return nil
	}

	if bg, err := NewBackgroundService(r); err != nil {
		return nil
	} else {
		r.bg = bg
	}

	r.conn = conn
	r.mbf = mf.NewBufferFactory()
	r.maxlen = 4096

	r.in = in
	r.wrapper = wrapper

	r.b = make([]byte, r.maxlen+r.mbf.GetHdrLen())

	if logger == nil {
		r.logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		r.logger = logger
	}

	return r
}

func (r *Reader) Run() {
	r.bg.Run()
}

func (r *Reader) Stop() {
	r.bg.Stop()
}

func (r *Reader) ServiceLoop(q chan struct{}, ready chan bool) {
	ready <- true
forever:
	for {
		if p, err := r.read(); err != nil {
			// TODO: timeout or error?
			r.err = err
			break forever
		} else {
			select {
			case r.in <- r.wrapper.wrap(p):
			case <-q:
				// TODO: cleanup
				break forever
			}
		}
	}
}

//Try to read a whole msg.
func (r *Reader) read() (Payload, error) {
	hb := r.b[0:r.mbf.GetHdrLen()]
	// TODO: Read() interrupt?
	if _, err := r.conn.Read(hb); err != nil {
		return nil, err
	}

	if err := r.mbf.UnmarshalHeader(hb); err != nil {
		return nil, err
	}

	plen := r.mbf.GetPayloadLen()
	pb := r.b[r.mbf.GetHdrLen() : r.mbf.GetHdrLen()+plen]
	// Support header-only message(udp or no payload at all).
	if plen > 0 {
		// TODO: enlarge b []byte if plen > r.maxlen or error out.
		if n, err := r.conn.Read(pb); err != nil {
			return nil, err
		} else if uint32(n) != plen {
			// TODO: error?
		}
	}

	p, err := r.mbf.UnmarshalPayload(pb)
	if err != nil {
		return nil, err
	}

	return p, nil
}
