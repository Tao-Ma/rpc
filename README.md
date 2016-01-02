# rpc

This is a simple rpc framework.

Architecture
___

	Router
		|-EndPoint
			|-Reader
			|-Writer
		|-Listern

TODO
___
- Compatibility: 'grpc' register server feature
- Featue: Support Reader/Writer timeout(Defer/Deadline)
- Featue: Error
- Featue: Log
- Featue: Route Rule
- Test: Call timeout
- Performance: Reuse EndPoint
- Performance: Reuse Reader/Writer
- Performance: Task Queue(May not useful, gc trace show there is no gc operation at all.)
- Performance: Use lower level event module/shareReader/shareWriter
- Security: transport
- Protocol: Http Message
- Protocol: Http 2.0
- Compatibility: grpc api
- Compatibility: grpc stream
