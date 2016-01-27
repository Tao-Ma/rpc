# rpc

This is a simple rpc framework. It is my first GO project.

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
- BUG: It looks like huge concurrent requests with small buffered reader/writer will cause crash.
- Feature: EndPoint Notify(client close Router and report error.)
- Feature: writer timeout using time.Tick instead of using time.After
- Feature: Support Reader/Writer timeout(Defer/Deadline)
- Feature: Support Large Message.
- Feature: Error
- Feature: Log
- Feature: Route Rule
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

Performance
---
Very fast and the throughput are quite high. On my 2014 Mid rmbp 15, more
than 200k qps(latency < 50ms) with 8 connections 500 concurrent request.

14+k qps(avg latency < 70us) with 1 concurrent request.
31+k qps(avg latency < 65us) with 2 concurrent request.
41+k qps(avg latency < 75us) with 3 concurrent request.
47+k qps(avg latency < 85us) with 4 concurrent request.
54+k qps(avg latency < 110us) with 6 concurrent request.
60+k qps(avg latency < 140us) with 8 concurrent request.
120+k qps(avg latency < 1ms) with 100 concurrent request.
120+k qps(avg latency < 5ms) with 500 concurrent request.

Performance are a little better than grpc(grpc support more features, I hope
I can run faster when I support all of such features).
