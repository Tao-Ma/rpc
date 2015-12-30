// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"io"
	"log"
	"os"
)

var (
	ErrWriterClosed error = &Error{err: "Writer closed"}
)

type PayloadChan struct {
	ch chan Payload

	owner *ResourceManager
}

func NewPayloadChan(ch chan Payload) *PayloadChan {
	pc := new(PayloadChan)
	pc.ch = ch
	return pc
}

func (pc *PayloadChan) Reset() {
	// do nothing
}

func (pc *PayloadChan) Recycle() {
	pc.owner.Put(pc)
}

func (pc *PayloadChan) SetOwner(o *ResourceManager) Resource {
	pc.owner = o
	return pc
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

	w.rm = NewResourceManager(128, func() Resource { return NewPayloadChan(io.Out()) })

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
		if w.err = w.LoopOnce(q); w.err != nil {
			// Remeber the last error
			w.io.OutError(w.err)
			break
		}
	}
}

func (w *Writer) Write(p Payload) error {
	ch := w.rm.Get().(*PayloadChan)
	if ch == nil {
		return ErrWriterClosed
	}

	select {
	case ch.ch <- p:
	default:
		panic("channel ref leaks?!")
	}

	ch.Recycle()

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
