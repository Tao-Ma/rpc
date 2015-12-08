// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

type MsgPayload interface {
	GetMsgPayloadID() uint16
}

type MsgPayloadBufferFactory interface {
	New(uint16) MsgPayload

	Marshal(MsgPayload, []byte) ([]byte, error)
	Unmarshal(uint16, []byte) (MsgPayload, error)
}

type MsgPayloadFactory interface {
	NewBufferFactory() MsgPayloadBufferFactory
}

type msgHeader struct {
	payload_id     uint16
	reserve        uint16
	length         uint32
	version        uint16
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

func (hf *MsgHeaderFactory) NewBufferFactory() MsgBufferFactory {
	hbf := new(msgHeaderBufferFactory)

	hbf.h.version = 1
	hbf.hdrlen = 16

	hbf.pbf = hf.pf.NewBufferFactory()

	return MsgBufferFactory(hbf)
}

type msgHeaderBufferFactory struct {
	h      msgHeader
	hdrlen uint32
	pbf    MsgPayloadBufferFactory
}

func (hbf *msgHeaderBufferFactory) GetHdrLen() uint32 {
	return hbf.hdrlen
}

func (hbf *msgHeaderBufferFactory) GetPayloadLen() uint32 {
	return hbf.h.length - hbf.hdrlen
}

func (hbf *msgHeaderBufferFactory) MarshalPayload(p Payload, b []byte) ([]byte, error) {
	mp, ok := p.(MsgPayload)
	if !ok {
		return b, nil
	}

	return hbf.pbf.Marshal(mp, b)
}

func (hbf *msgHeaderBufferFactory) UnmarshalPayload(b []byte) (Payload, error) {
	return hbf.pbf.Unmarshal(hbf.h.payload_id, b)
}

func (hbf *msgHeaderBufferFactory) MarshalHeader(b []byte, p Payload, l uint32) error {
	if uint32(len(b)) < hbf.hdrlen {
		return nil
	}
	mp, ok := p.(MsgPayload)
	if !ok {
		return nil
	}

	// Set the payload_id
	hbf.h.payload_id = mp.GetMsgPayloadID()
	// Set payload length
	hbf.h.length = hbf.hdrlen + l
	hbf.h.payload_offset = uint16(hbf.hdrlen)

	// TODO: employ a better pack/unpack method!
	off := 0

	b[off] = byte(hbf.h.payload_id >> 8)
	b[off+1] = byte(hbf.h.payload_id)
	off += 2

	// skip reserver
	off += 2

	b[off] = byte(hbf.h.length >> 24)
	b[off+1] = byte(hbf.h.length >> 16)
	b[off+2] = byte(hbf.h.length >> 8)
	b[off+3] = byte(hbf.h.length)
	off += 4

	b[off] = byte(hbf.h.version >> 8)
	b[off+1] = byte(hbf.h.version)
	off += 2

	b[off] = byte(hbf.h.payload_offset >> 8)
	b[off+1] = byte(hbf.h.payload_offset)
	off += 2

	b[off] = byte(hbf.h.checksum >> 24)
	b[off+1] = byte(hbf.h.checksum >> 16)
	b[off+2] = byte(hbf.h.checksum >> 8)
	b[off+3] = byte(hbf.h.checksum)
	off += 4

	return nil
}

func (hbf *msgHeaderBufferFactory) UnmarshalHeader(b []byte) error {
	var off uint32
	if uint32(len(b)) < hbf.hdrlen {
		return nil
	}

	hbf.h.payload_id = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	// skip reserve
	off += 2
	hbf.h.length = uint32(b[off]<<24) | uint32(b[off+1]<<16) | uint32(b[off+2]<<8) | uint32(b[off+3])
	off += 4
	hbf.h.version = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hbf.h.payload_offset = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	hbf.h.checksum = uint32(b[off]<<24) | uint32(b[off+1]<<16) | uint32(b[off+2]<<8) | uint32(b[off+3])
	off += 4

	// TODO: Validate() error
	return nil
}
