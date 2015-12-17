// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"io"
	"log"
	"os"
	"time"
)

var ()

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

	maxlen uint32
	b      []byte

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
	w.conn.Close()
}

func (w *Writer) Loop(q chan struct{}) {
forever:
	for {
		// forward msg from chan to conn
		select {
		case <-q:
			break forever
		case p := <-w.io.Out():
			// TODO: timeout or error?
			if b, err := w.write(p); err != nil {
				// TODO: error?
				panic(err)
			} else if _, err := w.conn.Write(b); err != nil {
				w.err = err
				// TODO: stop the writer?
				break forever
			}
		}
	}
}

func (w *Writer) Write(p Payload) error {
	// TODO: get token
	// Detect channel closed by nil

	select {
	case w.io.Out() <- p:
	default:
		select {
		case w.io.Out() <- p:
		case <-time.Tick(0 * time.Second):
			// TODO: timeout
			return nil
		}
	}

	return nil
}

func (w *Writer) write(p Payload) ([]byte, error) {
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

type Reader struct {
	bg *BackgroudService

	conn io.ReadCloser
	mb   MsgBuffer
	err  error

	io IOChannel

	maxlen uint32
	b      []byte

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

	return r
}

func (r *Reader) Run() {
	r.bg.Run()
}

func (r *Reader) Stop() {
	r.bg.Stop()
}

func (r *Reader) StopLoop(force bool) {
	r.conn.Close()
}

func (r *Reader) Loop(q chan struct{}) {
forever:
	for {
		if p, err := r.read(); err != nil {
			// TODO: timeout or error?
			r.err = err
			break forever
		} else {
			select {
			case r.io.In() <- p:
			case <-q:
				// TODO: cleanup
				break forever
			}
		}
	}
}

//Try to read a whole msg.
func (r *Reader) read() (Payload, error) {
	r.mb.Reset()
	hb := r.b[0:r.mb.GetHdrLen()]
	// TODO: Read() interrupt?
	if _, err := r.conn.Read(hb); err != nil {
		return nil, err
	}

	if err := r.mb.UnmarshalHeader(hb); err != nil {
		return nil, err
	}

	plen := r.mb.GetPayloadLen()
	pb := r.b[r.mb.GetHdrLen() : r.mb.GetHdrLen()+plen]
	// Support header-only message(udp or no payload at all).
	if plen > 0 {
		// TODO: enlarge b []byte if plen > r.maxlen or error out.
		if n, err := r.conn.Read(pb); err != nil {
			return nil, err
		} else if uint32(n) != plen {
			// TODO: error?
		}
	}

	p, err := r.mb.UnmarshalPayload(pb)
	if err != nil {
		return nil, err
	}
	if p == nil {
		panic("unmarshal failed")
	}

	p = r.io.Wrap(p)
	r.mb.GetPayloadInfo(p)

	return p, nil
}
