// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"io"
	"log"
	"os"
)

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
			r.io.InError(r.err)
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
