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
	maxlen         int
	b              []byte
	ob             []byte // always ref to w.b
	b_alloc_offset int
	b_data_offset  int
	buffered       bool
	tch            <-chan time.Time
	timeout        time.Duration

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
	w.b = make([]byte, w.maxlen*2)
	w.ob = w.b
	w.timeout = 10 * time.Microsecond

	w.buffered = false

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

func (w *Writer) ShouldFlush() bool {
	aoff := w.b_alloc_offset
	doff := w.b_data_offset
	len := aoff - doff

	return len >= w.maxlen || doff >= w.maxlen || !w.buffered
}

func (w *Writer) jitBuf() bool {
	return &w.ob[0] != &w.b[0]
}

func (w *Writer) write(force bool) error {
	if !force && !w.ShouldFlush() && !w.jitBuf() {
		return nil
	}

	aoff := w.b_alloc_offset
	doff := w.b_data_offset
	//w.logger.Printf("Writer(%v).write doff: %v aoff: %v len: %v\n", &w.bg, doff, aoff, aoff-doff)

	n, err := w.conn.Write(w.b[doff:aoff])
	if n < 0 {
		return err
	}

	w.b_data_offset += n
	if n != aoff-doff {
		// TODO: why?
		return err
	}

	return nil
}

func (w *Writer) LoopOnce(q chan struct{}) error {
	var force bool

	for {
		select {
		case <-q:
			// TODO: flush
			return errQuit
		case <-w.tch:
			// timeout
			w.tch = nil
			force = true
		case p := <-w.io.Out():
			if p == nil {
				return errQuit
			}
			if err := w.Marshal(p); err != nil {
				return err
			}

			// If there is no timer, add one.
			if w.buffered && w.tch == nil {
				w.tch = time.After(w.timeout)
			}

			if !w.ShouldFlush() {
				continue
			}
		}

		break
	}

	return w.write(force)
}

/*
func (w *Writer) LoopOnce(q chan struct{}) error {
	var force bool

	select {
	case <-q:
		// TODO: flush
		return errQuit
	case <-w.tch:
		// timeout
		w.tch = nil
		force = true
	case p := <-w.io.Out():
		if p == nil {
			return errQuit
		}
		if err := w.Marshal(p); err != nil {
			return err
		}

		// If there is no timer, add one.
		if w.tch == nil {
			w.tch = time.After(w.timeout)
		}

		force = !w.buffered || w.ShouldFlush()
	}

	return w.write(force)
}
*/

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

func (w *Writer) allocBuf(rlen uint32) []byte {
	len := int(rlen)

	aoff := w.b_alloc_offset
	doff := w.b_data_offset
	if doff >= w.maxlen {
		// rewind first
		copy(w.b[0:aoff-doff], w.b[doff:aoff])
		w.b_alloc_offset = aoff - doff
		w.b_data_offset = 0
		aoff = aoff - doff
	}

	if len == 0 {
		// don't change aoff
		return w.b[aoff:]
	}

	w.b_alloc_offset += len
	return w.b[aoff : aoff+len]
}

func (w *Writer) Marshal(p Payload) error {
	w.mb.Reset()

	w.mb.SetPayloadInfo(p)
	p = w.io.Unwrap(p)

	hb := w.allocBuf(w.mb.GetHdrLen())
	pb := w.allocBuf(0)

	npb, err := w.mb.MarshalPayload(p, pb)
	if err != nil {
		return err
	}

	if err := w.mb.MarshalHeader(hb, p, uint32(len(npb))); err != nil {
		return err
	}

	if &npb[0] == &pb[0] {
		// unchanged
		w.allocBuf(uint32(len(npb)))
		return nil
	}

	// changed
	w.b = append(w.b[w.b_data_offset:w.b_alloc_offset], npb...)
	w.b_data_offset = 0
	w.b_alloc_offset = len(w.b)
	return nil
}
