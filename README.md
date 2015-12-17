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
- Seperate BackgroundService to a standalone file
- Defer socket/listener/...
- Concurrent connect/close support
- Add token access for all writtable channel
