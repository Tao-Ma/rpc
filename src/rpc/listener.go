// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"net"
)

type Listener struct {
	name string
	l    net.Listener

	mf MsgFactory

	r     *Router
	serve ServeConn

	bg *BackgroudService
}

func NewListener(name string, listener net.Listener, mf MsgFactory, r *Router, serve ServeConn) (*Listener, error) {
	l := new(Listener)

	if bg, err := NewBackgroundService(l); err != nil {
		return nil, err
	} else {
		l.bg = bg
	}

	l.name = name
	l.l = listener
	l.mf = mf

	// TODO: change this to interface
	l.r = r
	l.serve = serve

	return l, nil
}

func (l *Listener) Run() {
	l.bg.Run()
}

func (l *Listener) Stop() {
	l.bg.Stop()
}

func (l *Listener) StopLoop(force bool) {
	l.l.Close()
}

func (l *Listener) Loop(q chan struct{}) {
	go l.accepter()

	select {
	case <-q:
		l.StopLoop(false)
	}
	// TODO: wait ?
}

func (l *Listener) Cleanup() {
	// nothing to do
}

func (l *Listener) accepter() {
	for {
		if c, err := l.l.Accept(); err != nil {
			// TODO: log?
			break
		} else if l.serve != nil && l.serve(l.r, c) {
			// TODO: serve
		} else if ep := l.r.newRouterEndPoint(l.name+c.RemoteAddr().String(), c, l.mf); ep == nil {
			c.Close()
			break
		} else if err := l.r.AddEndPoint(ep); err != nil {
			ep.Stop()
		} else {
			//l.r.logger.Print(ep)
		}
	}
}
