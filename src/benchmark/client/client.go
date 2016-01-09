package main

import (
	testpb "benchmark/proto_pb_test"
	"flag"
	"fmt"
	proto "github.com/golang/protobuf/proto"
	"rpc"
	"sync"
	"time"
)

func ClientProcessReponseWaitGroup(p rpc.Payload, arg rpc.RPCCallback_arg, err error) {
	arg.(*sync.WaitGroup).Done()
	if err != nil {
		panic("Error")
	}
}
func main() {
	address := flag.String("address", ":10000", "benchmark server address")
	conn_num := flag.Uint64("conn_num", 1, "benchmark connection number")
	req_total_num := flag.Uint64("req_total_num", 1000000, "benchmark request number")
	req_burst_num := flag.Uint64("req_burst_num", 1, "benchmark burst number")

	flag.Parse()

	conn_req_times := *req_total_num / *conn_num
	burst_times := conn_req_times / *req_burst_num

	hf := rpc.NewRPCHeaderFactory(rpc.NewProtobufFactory())
	_ = hf

	r, err := rpc.NewRouter(nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	r.Run()
	defer r.Stop()

	if err := r.Dial("benchmark", "tcp", *address, hf); err != nil {
		fmt.Println(err)
		return
	}

	var conn_wg sync.WaitGroup

	start := time.Now()
	for i := uint64(0); i < *conn_num; i++ {
		conn_wg.Add(1)
		// connection
		go func(r *rpc.Router, conn_wg *sync.WaitGroup, burst_times uint64, burst_num uint64) {
			for i := uint64(0); i < burst_times; i++ {
				var burst_wg sync.WaitGroup
				for j := uint64(0); j < burst_num; j++ {
					burst_wg.Add(1)
					req := testpb.NewTestReq()
					req.Id = proto.Uint64(1)

					r.Call("benchmark", "rpc", req, ClientProcessReponseWaitGroup, &burst_wg, 0)
				}
				burst_wg.Wait()
			}
			conn_wg.Done()
		}(r, &conn_wg, burst_times, *req_burst_num)
	}
	conn_wg.Wait()
	stop := time.Now()

	total_time := stop.Sub(start)
	qps := float64(*req_total_num*1000*1000*1000) / float64(total_time)

	fmt.Printf("total time: %v total request: %v qps: %v\n",
		total_time, *req_total_num, qps)
}
