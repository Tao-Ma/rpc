// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"io"
	"log"
	"os"
	"time"
)

var (
	errQuit      error = &Error{err: "quit"}
	errShortRead error = &Error{err: "short read"}
)

// IOChannel
type IOChannel interface {
	In() chan Payload
	Out() chan Payload

	// Support decorate data
	Unwrap(Payload) Payload
	Wrap(Payload) Payload
}

/*
type Encoder interface {
	GetHdrLen() uint32
	GetPayloadLen() uint32

	MarshalHeader([]byte, Payload, uint32) error
	MarshalPayload(Payload, []byte) ([]byte, error)
}

type Decoder interface {
	GetHdrLen() uint32
	GetPayloadLen() uint32

	UnmarshalHeader([]byte) error
	UnmarshalPayload([]byte) (Payload, error)
}
*/

type MsgFactory interface {
	NewBuffer() MsgBuffer
}

type MsgBuffer interface {
	Reset()

	MarshalHeader([]byte, Payload, uint32) error
	UnmarshalHeader([]byte) error

	MarshalPayload(Payload, []byte) ([]byte, error)
	UnmarshalPayload([]byte) (Payload, error)

	SetPayloadInfo(Payload)
	GetPayloadInfo(Payload)

	GetHdrLen() uint32
	GetPayloadLen() uint32
}

type Writer struct {
	bg *BackgroudService

	conn io.WriteCloser
	mb   MsgBuffer
	err  error

	io IOChannel

	rm *ResourceManager

	// buffer cache
	maxlen uint32
	b      []byte

	// inprogress state
	inprogress_p Payload
	inprogress_b []byte

	logger *log.Logger
}

func NewWriter(conn io.WriteCloser, io IOChannel, mb MsgBuffer, logger *log.Logger) *Writer {
	w := new(Writer)

	if bg, err := NewBackgroundService(w); err != nil {
		return nil
	} else {
		w.bg = bg
	}

	w.conn = conn
	w.mb = mb

	w.io = io

	w.rm = NewResourceManager(128, func() interface{} { return io.Out() })

	w.maxlen = 4096
	w.b = make([]byte, w.mb.GetHdrLen()+w.maxlen)

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

func (w *Writer) StopLoop(force bool) {
	// TODO: Add a handler to do something before conn.Close().
	w.conn.Close()
}

func (w *Writer) write(b []byte) (int, error) {
	return w.conn.Write(b)
}

func (w *Writer) LoopOnce(q chan struct{}) error {
	var err error

	if w.inprogress_b == nil {
		select {
		case <-q:
			return errQuit
		case w.inprogress_p = <-w.io.Out():
		}
	}

	if w.inprogress_p == nil {
		// channel closed
		return errQuit
	} else if w.inprogress_b, err = w.Marshal(w.inprogress_p); err != nil {
		// reset w.inprogress_p
		w.inprogress_p = nil
		w.inprogress_b = nil
		return err
	}

	if _, err = w.write(w.inprogress_b); err != nil {
		w.inprogress_b = nil
		return err
	}

	w.inprogress_b = nil
	return nil
}

func (w *Writer) Loop(q chan struct{}) {
	for {
		if err := w.LoopOnce(q); err != nil {
			// Remeber the last error
			w.err = err
			break
		}
	}
}

func (w *Writer) Write(p Payload) error {
	// TODO: get token
	// Detect channel closed by nil

	ch := w.rm.Get().(chan Payload)
	if ch == nil {
		// TODO: closed
		return nil
	}

	select {
	case ch <- p:
	default:
		select {
		case ch <- p:
		case <-time.Tick(0 * time.Second):
			// TODO: timeout
			return nil
		}
	}

	w.rm.Put(ch)

	return nil
}

func (w *Writer) Marshal(p Payload) ([]byte, error) {
	w.mb.Reset()

	w.mb.SetPayloadInfo(p)
	p = w.io.Unwrap(p)

	pb, err := w.mb.MarshalPayload(p, w.b[w.mb.GetHdrLen():w.mb.GetHdrLen()])
	if err != nil {
		return nil, err
	}

	hb := w.b[0:w.mb.GetHdrLen()]
	if err := w.mb.MarshalHeader(hb, p, uint32(len(pb))); err != nil {
		return nil, err
	}

	b := append(hb, pb...)

	return b, nil
}

const (
	header_init = iota
	header_read
	header_unmarshal
	body_init
	body_read
	body_unmarshal
)

type Reader struct {
	bg *BackgroudService

	conn io.ReadCloser
	mb   MsgBuffer
	err  error

	io IOChannel

	maxlen uint32
	b      []byte

	step int
	p    Payload
	hb   []byte
	pb   []byte

	logger *log.Logger
}

func NewReader(conn io.ReadCloser, io IOChannel, mb MsgBuffer, logger *log.Logger) *Reader {
	r := new(Reader)

	if bg, err := NewBackgroundService(r); err != nil {
		return nil
	} else {
		r.bg = bg
	}

	r.conn = conn
	r.mb = mb
	r.maxlen = 4096

	r.io = io

	r.b = make([]byte, r.maxlen+r.mb.GetHdrLen())

	if logger == nil {
		r.logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		r.logger = logger
	}

	r.step = header_init

	return r
}

func (r *Reader) Read() (Payload, error) {
	// TODO: not implement
	return nil, nil
}

func (r *Reader) Run() {
	r.bg.Run()
}

func (r *Reader) Stop() {
	r.bg.Stop()
}

func (r *Reader) StopLoop(force bool) {
	// TODO: Add a handler to do something before conn.Close().
	r.conn.Close()
}

func (r *Reader) Loop(q chan struct{}) {
	for {
		if r.p == nil {
			r.p, r.err = r.LoopOnce(q)
		}

		if r.err != nil {
			break
		}

		select {
		case r.io.In() <- r.p:
			r.p = nil
		case <-q:
			break
		}
	}
}

func (r *Reader) read(b []byte) (int, error) {
	return r.conn.Read(b)
}

func (r *Reader) LoopOnce(q chan struct{}) (Payload, error) {
	for {
		switch r.step {
		case header_init:
			r.mb.Reset()
			r.hb = r.b[0:r.mb.GetHdrLen()]
			r.step = header_read
		case header_read:
			if _, err := r.read(r.hb); err != nil {
				return nil, err
			}
			r.step = header_unmarshal
		case header_unmarshal:
			if err := r.mb.UnmarshalHeader(r.hb); err != nil {
				// invalid message
				return nil, err
			}
			r.step = body_init
		case body_init:
			plen := r.mb.GetPayloadLen()
			if plen > 0 {
				r.pb = r.b[r.mb.GetHdrLen() : r.mb.GetHdrLen()+plen]
				r.step = body_read
			} else {
				// TODO: no payload ?
			}
		case body_read:
			// TODO: enlarge b []byte if plen > r.maxlen or error out.
			plen := len(r.pb)
			if n, err := r.read(r.pb); err != nil {
				return nil, err
			} else if n != plen {
				return n, errShortRead
			}
			r.step = body_unmarshal
		case body_unmarshal:
			if p, err := r.mb.UnmarshalPayload(r.pb); err != nil {
				// invalid message
				return nil, err
			} else {
				p = r.io.Wrap(p)
				r.mb.GetPayloadInfo(p)
				r.step = header_init
				return p, nil
			}
		}
	}
}
