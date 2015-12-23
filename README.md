# rpc

This is a simple rpc framework.

Architect
___

	Router
		|-EndPoint
			|-Reader
			|-Writer
		|-Listern

TODO
___
- Support 'grpc' register feature
- Error/log
- Defer socket/listener/...
- Reuse Reader/Writer/EndPoint
- Ensure opReq.ret is resuable and concurrent safe
