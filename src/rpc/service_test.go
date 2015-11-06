package rpc

import (
	"testing"
)

type crouter struct{}

func (r *crouter) RpcRequestRoute(req RpcRequest) {
}

func (r *crouter) RpcResponseRoute(resp RpcResponse, req RpcRequest) {
}

func (r *crouter) DefaultRoute(v interface{}) {
}

func (r *crouter) ErrorRoute(v interface{}, text string) {
}

type srouter struct{}

func (r *srouter) RpcRequestRoute(req RpcRequest) {
}

func (r *srouter) RpcResponseRoute(resp RpcResponse, req RpcRequest) {
}

func (r *srouter) DefaultRoute(v interface{}) {
}

func (r *srouter) ErrorRoute(v interface{}, text string) {
}

func TestEndPoint(t *testing.T) {
	addr := "localhost:10000"

	c, _ := newDefaultService(&crouter{})
	s, _ := newDefaultService(&srouter{})

	c.Run()
	s.Run()

	if err := s.ListenAndServe("tcp", addr); err != nil {
		t.Log(err)
		t.Fail()
	}
	if err := c.Dial("tcp", addr); err != nil {
		t.Log(err)
		t.Fail()
	}

	c.Stop()
	s.Stop()
}
