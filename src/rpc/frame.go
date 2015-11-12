// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"io"
	"log"
	"os"
	"time"
)

type Header interface {
	GetLen() uint32
	Marshal() ([]byte, error)
	MarshalI([]byte) error
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

type ReaderWrapper interface {
	wrap(Payload) Payload
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

	out chan Payload

	hdr    Header
	hdrlen uint32
	buf    []byte

	logger *log.Logger
}

func NewWriter(conn io.Writer, out chan Payload, hf HeaderFactory, pf PayloadFactory, logger *log.Logger) *Writer {
	w := new(Writer)

	if bg, err := NewBackgroundService(w); err != nil {
		return nil
	} else {
		w.bg = bg
	}

	w.conn = conn
	w.hf = hf
	w.pf = pf

	w.out = out

	w.hdr = hf.New()
	w.hdrlen = w.hdr.GetLen()
	w.buf = make([]byte, w.hdrlen, w.hdrlen)

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
	pb, err := p.MarshalPayload()
	if err != nil {
		return nil, err
	}

	w.hdr.SetPayloadId(p.GetPayloadId())
	w.hdr.SetPayloadLen(uint32(len(pb)))

	if err := w.hdr.MarshalI(w.buf); err != nil {
		return nil, err
	}

	b := append(w.buf, pb...)

	return b, nil
}

func (w *Writer) ServiceLoop(q chan bool, r chan bool) {
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
	hf   HeaderFactory
	pf   PayloadFactory
	err  error

	maxlen  uint32
	in      chan Payload
	wrapper ReaderWrapper

	// Speedup the read memory
	hdr    Header
	hdrlen uint32
	buf    []byte

	logger *log.Logger
}

func NewReader(conn io.Reader, in chan Payload, wrapper ReaderWrapper, hf HeaderFactory, pf PayloadFactory, logger *log.Logger) *Reader {
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
	r.hf = hf
	r.pf = pf
	r.maxlen = 4096

	r.in = in
	r.wrapper = wrapper

	r.hdr = r.hf.New()
	r.hdrlen = r.hdr.GetLen()
	r.buf = make([]byte, r.hdrlen, r.maxlen+r.hdrlen)

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

func (r *Reader) ServiceLoop(q chan bool, ready chan bool) {
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
	hb := r.buf[:r.hdrlen]
	// TODO: Read() interrupt?
	if _, err := r.conn.Read(hb); err != nil {
		return nil, err
	}

	if err := r.hdr.Unmarshal(hb); err != nil {
		return nil, err
	}

	plen := r.hdr.GetPayloadLen()
	pb := r.buf[r.hdrlen : r.hdrlen+plen]
	// Support header-only message(udp or no payload at all).
	if plen > 0 {
		// TODO: enlarge b []byte if plen > r.maxlen or error out.
		if n, err := r.conn.Read(pb); err != nil {
			return nil, err
		} else if uint32(n) != plen {
		}
	}

	p := r.pf.New(r.hdr.GetPayloadId())
	if err := p.UnmarshalPayload(pb); err != nil {
		return nil, err
	}

	return p, nil
}
