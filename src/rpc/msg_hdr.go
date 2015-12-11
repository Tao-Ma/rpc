// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

type MsgPayload interface {
	Payload
	GetMsgPayloadID() uint16
}

type MsgPayloadBuffer interface {
	Marshal(MsgPayload, []byte) ([]byte, error)
	Unmarshal(uint16, []byte) (MsgPayload, error)
}

type MsgPayloadFactory interface {
	NewBuffer() MsgPayloadBuffer
}

const (
	MSG_RPC = 1 << iota
	MSG_REQUEST
)

type msgHeader struct {
	length         uint32
	rpcid          uint64
	version        uint16
	flags          uint16
	payload_id     uint16
	payload_offset uint16
	checksum       uint32
}

type MsgHeaderFactory struct {
	pf MsgPayloadFactory
}

func NewMsgHeaderFactory(pf MsgPayloadFactory) MsgFactory {
	hf := new(MsgHeaderFactory)
	hf.pf = pf
	return MsgFactory(hf)
}

func (hf *MsgHeaderFactory) NewBuffer() MsgBuffer {
	hb := new(msgHeaderBuffer)

	hb.h.version = 1
	hb.hdrlen = 24

	hb.b = hf.pf.NewBuffer()

	return MsgBuffer(hb)
}

type msgHeaderBuffer struct {
	h      msgHeader
	hdrlen uint32
	b      MsgPayloadBuffer
}

func (hb *msgHeaderBuffer) GetHdrLen() uint32 {
	return hb.hdrlen
}

func (hb *msgHeaderBuffer) GetPayloadLen() uint32 {
	return hb.h.length - hb.hdrlen
}

func (hb *msgHeaderBuffer) MarshalPayload(p Payload, b []byte) ([]byte, error) {
	mp, ok := p.(MsgPayload)
	if !ok {
		return b, nil
	}

	return hb.b.Marshal(mp, b)
}

func (hb *msgHeaderBuffer) UnmarshalPayload(b []byte) (Payload, error) {
	return hb.b.Unmarshal(hb.h.payload_id, b)
}

func (hb *msgHeaderBuffer) SetPayloadInfo(p Payload) {
	if rp, ok := p.(RoutePayload); !ok {
		return
	} else if !rp.IsRPC() {
		return
	} else {
		hb.h.flags |= MSG_RPC
		i := p.(RPCInfo)
		hb.h.rpcid = i.GetRPCID()
		if i.IsRequest() {
			hb.h.flags |= MSG_REQUEST
		}
	}
}

func (hb *msgHeaderBuffer) GetPayloadInfo(p Payload) {
	if rp, ok := p.(RoutePayload); !ok {
		return
	} else if (hb.h.flags & MSG_RPC) == 0 {
		return
	} else {
		rp.SetIsRPC()
		i := p.(RPCInfo)
		i.SetRPCID(hb.h.rpcid)
		if (hb.h.flags & MSG_REQUEST) == MSG_REQUEST {
			i.SetIsRequest()
		}
	}
}

func (hb *msgHeaderBuffer) MarshalHeader(b []byte, p Payload, l uint32) error {
	if uint32(len(b)) < hb.hdrlen {
		return nil
	}
	mp, ok := p.(MsgPayload)
	if !ok {
		return nil
	}

	// Set the payload_id
	hb.h.payload_id = mp.GetMsgPayloadID()
	// Set payload length
	hb.h.payload_offset = uint16(hb.hdrlen)
	// Set length
	hb.h.length = hb.hdrlen + l

	// TODO: employ a better pack/unpack method!
	off := 0

	b[off] = byte(hb.h.length >> 24)
	b[off+1] = byte(hb.h.length >> 16)
	b[off+2] = byte(hb.h.length >> 8)
	b[off+3] = byte(hb.h.length)
	off += 4

	b[off] = byte(hb.h.rpcid >> 56)
	b[off+1] = byte(hb.h.rpcid >> 48)
	b[off+2] = byte(hb.h.rpcid >> 40)
	b[off+3] = byte(hb.h.rpcid >> 32)
	b[off+4] = byte(hb.h.rpcid >> 24)
	b[off+5] = byte(hb.h.rpcid >> 16)
	b[off+6] = byte(hb.h.rpcid >> 8)
	b[off+7] = byte(hb.h.rpcid)
	off += 8

	b[off] = byte(hb.h.version >> 8)
	b[off+1] = byte(hb.h.version)
	off += 2

	b[off] = byte(hb.h.flags >> 8)
	b[off+1] = byte(hb.h.flags)
	off += 2

	b[off] = byte(hb.h.payload_id >> 8)
	b[off+1] = byte(hb.h.payload_id)
	off += 2

	b[off] = byte(hb.h.payload_offset >> 8)
	b[off+1] = byte(hb.h.payload_offset)
	off += 2

	b[off] = byte(hb.h.checksum >> 24)
	b[off+1] = byte(hb.h.checksum >> 16)
	b[off+2] = byte(hb.h.checksum >> 8)
	b[off+3] = byte(hb.h.checksum)
	off += 4

	return nil
}

func (hb *msgHeaderBuffer) UnmarshalHeader(b []byte) error {
	if uint32(len(b)) < hb.hdrlen {
		return nil
	}

	off := 0

	hb.h.length = uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
	off += 4

	hb.h.rpcid = (uint64(b[off])<<24 | uint64(b[off+1])<<16 | uint64(b[off+2])<<8 | uint64(b[off+3])) << 32
	off += 4
	hb.h.rpcid |= uint64(b[off])<<24 | uint64(b[off+1])<<16 | uint64(b[off+2])<<8 | uint64(b[off+3])
	off += 4

	hb.h.version = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.flags = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.payload_id = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.payload_offset = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.checksum = uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
	off += 4
	// TODO: Validate() error

	return nil
}

func (hb *msgHeaderBuffer) Reset() {
	hb.h.length = 0
	hb.h.rpcid = 0
	hb.h.version = 1
	hb.h.flags = 0
	hb.h.payload_id = 0
	hb.h.payload_offset = 0
	hb.h.checksum = 0
}
