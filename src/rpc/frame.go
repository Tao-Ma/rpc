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

type ReaderDispatch interface {
	Dispatch(interface{})
}

type HeaderFactory interface {
	New() Header
}

type PayloadFactory interface {
	New(id uint16) Payload
}

type Writer struct {
	running bool
	quit    chan bool
	err     error

	conn io.Writer
	hf   HeaderFactory
	pf   PayloadFactory

	out chan *[]byte
}

func NewWriter(conn io.Writer, hf HeaderFactory, pf PayloadFactory) *Writer {
	w := new(Writer)

	w.quit = make(chan bool)
	w.conn = conn
	w.hf = hf
	w.pf = pf

	w.out = make(chan *[]byte)

	return w
}

func (w *Writer) Run() {
	if w.running {
		return
	}
	go w.loop()
}

func (w *Writer) Stop() {
	if !w.running {
		return
	}
	w.running = false
	w.quit <- true
	close(w.quit)
	// TODO: cleanup
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
	case <-time.Tick(time.Second):
		// timeout case
		return nil
	}

	// TODO: block support
	return nil
}

func (w *Writer) loop() {
forever:
	for {
		// forward msg from chan to conn
		select {
		case <-w.quit:
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
	running bool
	quit    chan bool
	err     error

	conn   io.Reader
	hf     HeaderFactory
	pf     PayloadFactory
	maxlen uint32

	ctx interface{}
}

func NewReader(conn io.Reader, hf HeaderFactory, pf PayloadFactory, ctx interface{}) *Reader {
	r := new(Reader)

	r.quit = make(chan bool)
	r.conn = conn
	r.hf = hf
	r.pf = pf
	r.maxlen = 4096

	r.ctx = ctx

	return r
}

func (r *Reader) Run() {
	if r.running {
		return
	}
	go r.loop()
}

func (r *Reader) Stop() {
	if !r.running {
		return
	}
	r.running = false
	r.quit <- true
	close(r.quit)
	// TODO: cleanup
}

func (r *Reader) loop() {
forever:
	for {
		if err := r.read(); err != nil {
			// TODO: timeout or error?
			r.err = err
			break forever
		}

		select {
		case <-r.quit:
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

	if d, ok := p.(ReaderDispatch); !ok {
		return nil
	} else {
		go d.Dispatch(r.ctx)
	}
	return nil
}
