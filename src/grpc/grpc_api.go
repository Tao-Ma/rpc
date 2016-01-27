// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package grpc

import (
	"net"
	"rpc"
	pbt "rpc/pb_test"
)

type Server struct {
	r *rpc.Router

	ln string
	mf rpc.MsgFactory
}

type ServiceDesc struct {
}

type ServerOption func(*options)

type options struct{}
type Credentials interface{}
type Codec interface{}

func Creds(c Credentials) ServerOption {
	return func(o *options) {
		//o.creds = c
	}
}

func CustomCodec(codec Codec) ServerOption {
	return func(o *options) {
		//o.codec = codec
	}
}

func MaxConcurrentStreams(n uint32) ServerOption {
	return func(o *options) {
		//o.maxConcurrentStreams = n
	}
}

func NewServer(opt ...ServerOption) *Server {
	var err error
	s := new(Server)

	if s.r, err = rpc.NewRouter(nil, nil); err != nil {
		return nil
	}

	s.ln = "grpc-api-listener"
	s.mf = rpc.NewMsgHeaderFactory(pbt.NewMsgProtobufFactory())

	return s
}

func (s *Server) RegisterService(sd *ServiceDesc, ss interface{}) {
}

func (s *Server) Serve(lis net.Listener) error {
	// TODO: hang if Router does not Run.
	s.r.Run()

	l, err := rpc.NewListener(s.ln, lis, s.mf, s.r, nil)
	if err != nil {
		return err
	}
	err = s.r.AddListener(l)
	if err != nil {
		l.Stop()
		return err
	}

	return nil
}

func (s *Server) Stop() {
	s.r.Stop()
}

/*
 * Client
 */
type ClientConn struct {
	r *rpc.Router

	cn string
	mf rpc.MsgFactory
}

type dialOptions struct {
}

type DialOption func(*dialOptions)

func Dial(target string, opts ...DialOption) (*ClientConn, error) {
	var err error
	cc := new(ClientConn)

	if cc.r, err = rpc.NewRouter(nil, nil); err != nil {
		return nil, err
	}

	cc.r.Run()

	cc.cn = "grpc-api-connector"
	cc.mf = rpc.NewMsgHeaderFactory(pbt.NewMsgProtobufFactory())

	if err = cc.r.Dial(cc.cn, "tcp", target, cc.mf); err != nil {
		cc.r.Stop()
		return nil, err
	}

	return cc, nil
}

func (cc *ClientConn) Close() error {
	cc.r.Stop()
	return nil
}

// TODO: func (cc *ClientConn) State() (ConnectivityState, error)
// TODO: func (cc *ClientConn) WaitForStateChange(ctx context.Context, sourceState ConnectivityState) (ConnectivityState, error)
