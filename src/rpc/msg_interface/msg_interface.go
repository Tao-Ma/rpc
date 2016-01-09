// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package msg_interface

type Payload interface {
}

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
