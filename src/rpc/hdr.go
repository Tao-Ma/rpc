// Copyright (C) Tao Ma(tao.ma.1984@gmail.com)

package rpc

import ()

type DefaultHeader struct {
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

func (hf *DefaultHeaderFactory) New() Header {
	return NewHeader()
}

func NewHeader() Header {
	h := new(DefaultHeader)

	h.version = 1
	return Header(h)
}

func (h *DefaultHeader) GetLen() uint32 {
	return 16
}

func (h *DefaultHeader) SetPayloadId(id uint16) {
	h.payload_id = id
}

func (h *DefaultHeader) GetPayloadId() uint16 {
	return h.payload_id
}

func (h *DefaultHeader) SetPayloadLen(l uint32) {
	h.length = h.GetLen() + l
	h.payload_offset = uint16(h.GetLen())
}

func (h *DefaultHeader) GetPayloadLen() uint32 {
	return h.length - h.GetLen()
}

func (h *DefaultHeader) Marshal() (b []byte, err error) {
	b = make([]byte, h.GetLen())

	h.checksum = 0

	// TODO: employ a better pack/unpack method!
	off := 0

	b[off] = byte(h.payload_id >> 8)
	b[off+1] = byte(h.payload_id)
	off += 2

	// skip reserver
	off += 2

	b[off] = byte(h.length >> 24)
	b[off+1] = byte(h.length >> 16)
	b[off+2] = byte(h.length >> 8)
	b[off+3] = byte(h.length)
	off += 4

	b[off] = byte(h.version >> 8)
	b[off+1] = byte(h.version)
	off += 2

	b[off] = byte(h.payload_offset >> 8)
	b[off+1] = byte(h.payload_offset)
	off += 2

	b[off] = byte(h.checksum >> 24)
	b[off+1] = byte(h.checksum >> 16)
	b[off+2] = byte(h.checksum >> 8)
	b[off+3] = byte(h.checksum)
	off += 4

	return b, nil
}

func (h *DefaultHeader) Unmarshal(b []byte) error {
	var off uint32

	if uint32(len(b)) < h.GetLen() {
		panic("")
		return nil // &Error{err: "invalid header size"}
	}

	h.payload_id = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	// skip reserve
	off += 2
	h.length = uint32(b[off]<<24) | uint32(b[off+1]<<16) | uint32(b[off+2]<<8) | uint32(b[off+3])
	off += 4
	h.version = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	h.payload_offset = uint16(b[off])<<8 | uint16(b[off+1])
	off += 2
	h.checksum = uint32(b[off]<<24) | uint32(b[off+1]<<16) | uint32(b[off+2]<<8) | uint32(b[off+3])
	off += 4

	// TODO: Validate() error
	return nil
}
