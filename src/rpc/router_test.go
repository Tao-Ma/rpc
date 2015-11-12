// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
	"net"
	"testing"
	"time"
)

func (r *ResourceReq) RpcGetId() uint64 {
	return r.GetId()
}

func (r *ResourceReq) RpcSetId(id uint64) {
	r.Id = proto.Uint64(id)
}

func (r *ResourceReq) RpcIsRequest() bool {
	return true
}

func (r *ResourceReq) RpcIsResponse() bool {
	return false
}

func (r *ResourceResp) RpcGetId() uint64 {
	return r.GetId()
}

func (r *ResourceResp) RpcSetId(id uint64) {
	r.Id = proto.Uint64(id)
}

func (r *ResourceResp) RpcIsRequest() bool {
	return false
}

func (r *ResourceResp) RpcIsResponse() bool {
	return true
}

func ServiceProcessConn(r *Router, c net.Conn) bool {
	return false
	/*
		ep := r.newHijackedEndPoint("hijacked", c, r.hf, r.pf)

		ep.Read()
		ep.Write()
		<-time.Tick(1 * time.Second)

		ep.Close()
		return true
	*/
}

func ServiceProcessPayload(r *Router, name string, p Payload) bool {
	resp := NewResourceResp()
	req := p.(*ResourceReq)
	resp.Id = proto.Uint64(req.GetId())

	r.Write(name, resp)
	return true
}

func ClientProcessReponse(p Payload, arg rpc_arg, err error) {
	done := arg.(chan bool)
	done <- true
}

func ClientProcessReponseIgnore(p Payload, arg rpc_arg, err error) {
}

func TestRouter(t *testing.T) {
	r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		t.FailNow()
	}

	hf := NewDefaultHeaderFactory()
	pf := NewProtobufFactory()

	name := "scheduler"
	network := "tcp"
	address := "localhost:10000"

	r.Run()

	if err := r.ListenAndServe("client", network, address, hf, pf, ServiceProcessConn); err != nil {
		t.Log(err)
		t.FailNow()
	}
	if err := r.Dial(name, network, address, hf, pf); err != nil {
		t.Log(err)
		t.FailNow()
	}

	req := NewResourceReq()
	/*
		if resp := r.CallWait(name, req, 0); resp == nil {
			//t.Log("CallWait timeout")
			//t.FailNow()
		}
	*/

	done := make(chan bool)
	r.Call("scheduler", req, ClientProcessReponse, done)
	<-done

	r.DelEndPoint("scheduler")

	r.DelListener("client")

	r.Stop()
}

func TestReadWriter(t *testing.T) {
	s, c := net.Pipe()

	ch_c_w := make(chan Payload, 1024)
	ch_s_w := make(chan Payload, 1024)
	ch_d := make(chan Payload, 1024)

	hf := NewDefaultHeaderFactory()
	pf := NewProtobufFactory()

	ep_c := NewEndPoint("c", c, ch_c_w, ch_d, nil, hf, pf, nil)
	ep_s := NewEndPoint("s", s, ch_s_w, ch_s_w, nil, hf, pf, nil)

	ep_c.Run()
	ep_s.Run()

	req := NewResourceReq()
	req.Id = proto.Uint64(1)
	ep_c.write(req)
	<-ch_d
}

func BenchmarkReadWriter(b *testing.B) {
	//s, c := net.Pipe()

	network := "tcp"
	address := "localhost:10008"

	l, err := net.Listen(network, address)
	if err != nil {
		b.FailNow()
	}
	c, err := net.Dial(network, address)
	if err != nil {
		b.FailNow()
	}
	s, err := l.Accept()
	if err != nil {
		b.FailNow()
	}

	ch_c_w := make(chan Payload, 1024)
	ch_s_w := make(chan Payload, 1024)
	ch_d := make(chan Payload, 1024)

	hf := NewDefaultHeaderFactory()
	pf := NewProtobufFactory()

	ep_c := NewEndPoint("c", c, ch_c_w, ch_d, nil, hf, pf, nil)
	ep_s := NewEndPoint("s", s, ch_s_w, ch_s_w, nil, hf, pf, nil)

	ep_c.Run()
	ep_s.Run()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := NewResourceReq()
		req.Id = proto.Uint64(1)
		for pb.Next() {
			ch_c_w <- req
			<-ch_d
		}
	})

}

func BenchmarkRouter(b *testing.B) {
	r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}

	hf := NewDefaultHeaderFactory()
	pf := NewProtobufFactory()

	name := "scheduler"
	/*
		network := "tcp"
		address := "localhost:10001"

		r.Run()
		<-time.Tick(1 * time.Millisecond)
		if err := r.ListenAndServe("client", network, address, hf, pf, ServiceProcessConn); err != nil {
			b.Log(err)
			b.FailNow()
		}
		if err := r.Dial(name, network, address, hf, pf); err != nil {
			b.Log(err)
			b.FailNow()
		}
	*/
	r.Run()
	<-time.Tick(1 * time.Millisecond)
	c, s := net.Pipe()
	ep_c := r.newRouterEndPoint(name, c, hf, pf)
	ep_s := r.newRouterEndPoint("client", s, hf, pf)
	r.AddEndPoint(ep_c)
	r.AddEndPoint(ep_s)

	<-time.Tick(1 * time.Millisecond)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := NewResourceReq()
			r.Call("scheduler", req, ClientProcessReponseIgnore, nil)
			//if resp := r.CallWait(name, req, 0); resp == nil {
			//		b.Log("CallWait timeout")
			//	b.FailNow()
			//}
		}
	})

	r.DelEndPoint("scheduler")

	r.DelListener("client")

	r.Stop()
}

func BenchmarkTcpSyncReadWriteSeperate(b *testing.B) {
	network := "tcp"
	address := "localhost:10009"

	l, err := net.Listen(network, address)
	if err != nil {
		b.FailNow()
	}
	c, err := net.Dial(network, address)
	if err != nil {
		b.FailNow()
	}
	s, err := l.Accept()
	if err != nil {
		b.FailNow()
	}

	ch_r := make(chan uint64, 1024)
	ch_w := make(chan uint64, 1024)
	ch_d := make(chan uint64, 1024)

	ch_c_w := make(chan uint64, 1024)
	ch_s_w := make(chan uint64, 1024)
	go func(s net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
			ch_r <- 3
		}
	}(s, ch_r)
	go func(s net.Conn, ch_s_w chan uint64) {
		for {
			<-ch_s_w
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s, ch_s_w)
	go func(c net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
			ch_r <- 5
		}
	}(c, ch_r)
	go func(c net.Conn, ch_c_w chan uint64) {
		for {
			<-ch_c_w
			bo := make([]byte, 19)
			c.Write(bo)
		}
	}(c, ch_c_w)
	go func(ch_r chan uint64, ch_w chan uint64, ch_d chan uint64, ch_c_w chan uint64, ch_s_w chan uint64) {
		for {
			select {
			case id := <-ch_r:
				if id == 3 {
					ch_s_w <- 4
				} else if id == 5 {
					ch_d <- 6
				}
			case <-ch_w:
				ch_c_w <- 2
			}
		}
	}(ch_r, ch_w, ch_d, ch_c_w, ch_s_w)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch_w <- 1
			<-ch_d
		}
	})
}

func BenchmarkPipeSyncReadWriteSeperate(b *testing.B) {
	s, c := net.Pipe()

	ch_r := make(chan uint64, 1024)
	ch_w := make(chan uint64, 1024)
	ch_d := make(chan uint64, 1024)

	ch_c_w := make(chan uint64, 1024)
	ch_s_w := make(chan uint64, 1024)
	go func(s net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
			ch_r <- 3
		}
	}(s, ch_r)
	go func(s net.Conn, ch_s_w chan uint64) {
		for {
			<-ch_s_w
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s, ch_s_w)
	go func(c net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
			ch_r <- 5
		}
	}(c, ch_r)
	go func(c net.Conn, ch_c_w chan uint64) {
		for {
			<-ch_c_w
			bo := make([]byte, 19)
			c.Write(bo)
		}
	}(c, ch_c_w)
	go func(ch_r chan uint64, ch_w chan uint64, ch_d chan uint64, ch_c_w chan uint64, ch_s_w chan uint64) {
		for {
			select {
			case id := <-ch_r:
				if id == 3 {
					ch_s_w <- 4
				} else if id == 5 {
					ch_d <- 6
				}
			case <-ch_w:
				ch_c_w <- 2
			}
		}
	}(ch_r, ch_w, ch_d, ch_c_w, ch_s_w)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch_w <- 1
			<-ch_d
		}
	})
}

func BenchmarkPipeShare(b *testing.B) {

	s, c := net.Pipe()
	go func(s net.Conn) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bo := make([]byte, 19)
			c.Write(bo)
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
		}
	})
}

func BenchmarkTcpShare(b *testing.B) {
	network := "tcp"
	address := "localhost:10003"

	l, err := net.Listen(network, address)
	if err != nil {
		b.FailNow()
	}
	c, err := net.Dial(network, address)
	if err != nil {
		b.FailNow()
	}
	s, _ := l.Accept()
	go func(s net.Conn) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bo := make([]byte, 19)
			c.Write(bo)
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
		}
	})
}

func BenchmarkTcpReadWriteSeperate(b *testing.B) {
	network := "tcp"
	address := "localhost:10005"

	l, err := net.Listen(network, address)
	if err != nil {
		b.FailNow()
	}
	c, err := net.Dial(network, address)
	if err != nil {
		b.FailNow()
	}
	s, _ := l.Accept()
	go func(s net.Conn) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
		}
	}(s)
	go func(s net.Conn) {
		for {
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bo := make([]byte, 19)
			c.Write(bo)
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
		}
	})
}

func BenchmarkChan(b *testing.B) {
	ch := make(chan uint64, 10240)

	go func(ch chan uint64) {
		for {
			select {
			case ch <- 1:
			case <-ch:
			}
		}
	}(ch)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch <- 1
			<-ch
		}
	})
}
