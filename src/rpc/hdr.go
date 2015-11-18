// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import ()

type defaultHeader struct {
	payload_id     uint16
	reserve        uint16
	length         uint32
	version        uint16
	payload_offset uint16
	checksum       uint32
}

type DefaultHeaderFactory struct{}

func NewDefaultHeaderFactory() HeaderFactory {
	hf := new(DefaultHeaderFactory)
	return HeaderFactory(hf)
}

func (hf *DefaultHeaderFactory) NewBufferFactory() HeaderBufferFactory {
	hbf := new(defaultHeaderBufferFactory)

	hbf.h.version = 1
	hbf.hdrlen = 16

	return HeaderBufferFactory(hbf)
}

type defaultHeaderBufferFactory struct {
	h      defaultHeader
	hdrlen uint32
}

func (hbf *defaultHeaderBufferFactory) GetHdrLen() uint32 {
	return hbf.hdrlen
}

func (hbf *defaultHeaderBufferFactory) SetPayloadId(id uint16) {
	hbf.h.payload_id = id
}

func (hbf *defaultHeaderBufferFactory) GetPayloadId() uint16 {
	return hbf.h.payload_id
}

func (hbf *defaultHeaderBufferFactory) SetPayloadLen(l uint32) {
	hbf.h.length = hbf.hdrlen + l
	hbf.h.payload_offset = uint16(hbf.hdrlen)
}

func (hbf *defaultHeaderBufferFactory) GetPayloadLen() uint32 {
	return hbf.h.length - hbf.hdrlen
}

func (hbf *defaultHeaderBufferFactory) Marshal(b []byte) error {
	if uint32(len(b)) < hbf.hdrlen {
		return nil
	}

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

func (hbf *defaultHeaderBufferFactory) Unmarshal(b []byte) error {
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
