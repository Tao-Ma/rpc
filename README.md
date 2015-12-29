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
- Support 'grpc' register server feature
- Reuse EndPoint
- Reuse Reader/Writer
- Error
- Log
- Support Reader/Writer timeout(Defer/Deadline)
- Ensure opReq.ret is resuable and concurrent safe
- Test Call timeout
- Route Rule
- Performance: Use lower level event module/shareReader/shareWriter
- Security transport
- Http Message
- Http 2.0
- grpc compatible
