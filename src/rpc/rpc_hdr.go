package rpc

import ()

// RPCPayload
type RPCPayload interface {
	GetRPCPayloadName() string
}

type RPCPayloadBuffer interface {
	Marshal(RPCPayload, []byte) ([]byte, error)
	Unmarshal(string, []byte) (RPCPayload, error)
}

type RPCPayloadFactory interface {
	NewBuffer() RPCPayloadBuffer
}

// RPCHeader
type rpcHeader struct {
	length           uint32
	rpcid            uint64
	version          uint16
	rpc_name_len     uint16
	payload_name_len uint16
	payload_offset   uint16
	checksum         uint32
	flags            uint16

	/* variable part */
	rpc_name     string
	payload_name string
}

type RPCHeaderFactory struct {
	pf RPCPayloadFactory
}

func NewRPCHeaderFactory(pf RPCPayloadFactory) MsgFactory {
	hf := new(RPCHeaderFactory)
	hf.pf = pf
	return MsgFactory(hf)
}

func (hf *RPCHeaderFactory) NewBuffer() MsgBuffer {
	hb := new(rpcHeaderBuffer)

	hb.h.version = 1
	hb.hdrlen = 24

	hb.b = hf.pf.NewBuffer()

	return MsgBuffer(hb)
}

type rpcHeaderBuffer struct {
	h      rpcHeader
	hdrlen uint32
	b      RPCPayloadBuffer
}

func (hb *rpcHeaderBuffer) GetHdrLen() uint32 {
	return hb.hdrlen
}

/* The rpc_name and payload_name stores at the beginning of payload. */
func (hb *rpcHeaderBuffer) GetPayloadLen() uint32 {
	return hb.h.length - hb.hdrlen
}

func (hb *rpcHeaderBuffer) MarshalPayload(p Payload, b []byte) ([]byte, error) {
	rp, ok := p.(RPCPayload)
	if !ok {
		return b, nil
	}

	if b, err := hb.marshalHeaderVariable(b); err != nil {
		return nil, err
	} else {
		return hb.b.Marshal(rp, b)
	}
}

func (hb *rpcHeaderBuffer) UnmarshalPayload(b []byte) (Payload, error) {
	if b, err := hb.unmarshalHeaderVariable(b); err != nil {
		return nil, err
	} else {

		return hb.b.Unmarshal(hb.h.payload_name, b)
	}
}

func (hb *rpcHeaderBuffer) SetPayloadInfo(p Payload) {
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

func (hb *rpcHeaderBuffer) GetPayloadInfo(p Payload) {
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

func (hb *rpcHeaderBuffer) MarshalHeader(b []byte, p Payload, l uint32) error {
	if uint32(len(b)) < hb.hdrlen {
		return nil
	}
	rp, ok := p.(RPCPayload)
	if !ok {
		return nil
	}

	// TODO: Set rpc_name
	hb.h.rpc_name_len = uint16(len(hb.h.rpc_name))
	// Set payload_name
	hb.h.payload_name = rp.GetRPCPayloadName()
	hb.h.payload_name_len = uint16(len(hb.h.payload_name))
	// Set payload_offset
	hb.h.payload_offset = uint16(uint16(hb.hdrlen) + hb.h.rpc_name_len + hb.h.payload_name_len)

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

	b[off] = byte(hb.h.rpc_name_len >> 8)
	b[off+1] = byte(hb.h.rpc_name_len)
	off += 2

	b[off] = byte(hb.h.payload_name_len >> 8)
	b[off+1] = byte(hb.h.payload_name_len)
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

func (hb *rpcHeaderBuffer) marshalHeaderVariable(b []byte) ([]byte, error) {
	// Write rpc_name
	copy(b[0:], hb.h.rpc_name)

	// Write payload_name
	copy(b[hb.h.rpc_name_len:], hb.h.payload_name)

	return b[hb.h.payload_offset:], nil
}

func (hb *rpcHeaderBuffer) UnmarshalHeader(b []byte) error {
	if uint32(len(b)) < hb.hdrlen {
		return nil
	}

	off := 0
	hb.h.length = uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
	off += 4
	hb.h.rpcid = (uint64(b[off]<<24) | uint64(b[off+1])<<16 | uint64(b[off+2])<<8 | uint64(b[off+3])) << 32
	off += 4
	hb.h.rpcid |= uint64(b[off])<<24 | uint64(b[off+1])<<16 | uint64(b[off+2])<<8 | uint64(b[off+3])
	off += 4

	hb.h.version = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.rpc_name_len = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.payload_name_len = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.payload_offset = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hb.h.checksum = uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
	off += 4

	return nil
}

func (hb *rpcHeaderBuffer) unmarshalHeaderVariable(b []byte) ([]byte, error) {
	// Read rpc_name
	hb.h.rpc_name = string(b[0:hb.h.rpc_name_len])
	// Read payload_name
	hb.h.payload_name = string(b[hb.h.rpc_name_len:hb.h.payload_offset])
	return b[hb.h.payload_offset:], nil
}

func (hb *rpcHeaderBuffer) Reset() {
	hb.h.length = 0
	hb.h.rpcid = 0
	hb.h.version = 1
	hb.h.flags = 0
	hb.h.rpc_name_len = 0
	hb.h.payload_name_len = 0
	hb.h.payload_offset = 0
	hb.h.checksum = 0
	hb.h.rpc_name = ""
	hb.h.payload_name = ""
}
