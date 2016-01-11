package main

import (
	"benchmark"
	testpb "benchmark/proto_pb_test"
	"flag"
	"fmt"
	proto "github.com/golang/protobuf/proto"
	"rpc"
	"sync"
	"time"
)

func ClientProcessReponseWaitGroup(p rpc.Payload, arg rpc.RPCCallback_arg, err error) {
	tc := arg.(*TaskCall)
	tc.arg.(*sync.WaitGroup).Done()
	tc.stop = time.Now()

	if err != nil {
		panic("Error")
	}
}

type TaskCall struct {
	start time.Time
	stop  time.Time

	arg interface{}
}

type Task struct {
	// conn/burst/id
	conn_num  int
	burst_num int
	task      map[int]map[int]map[uint64]*TaskCall
}

func (t *Task) Do(c *benchmark.Collector) {
	for _, conn_task := range t.task {
		for _, burst_task := range conn_task {
			for _, tc := range burst_task {
				d := tc.stop.Sub(tc.start)
				// us(MicroSecond) uint
				c.Add(uint64(d) / uint64(1000))
			}
		}
	}
}

func NewTask(req_num uint64, conn_num uint64, burst_num uint64) *Task {
	t := new(Task)

	t.task = make(map[int]map[int]map[uint64]*TaskCall)

	per_conn_req_num := req_num / conn_num

	conn_id := 0
	burst_id := 0

	for id := uint64(0); id < req_num; id++ {
		// should advance conn_id
		conn_id = int(id/per_conn_req_num) + 1
		if _, exist := t.task[conn_id]; !exist {
			t.task[conn_id] = make(map[int]map[uint64]*TaskCall)
			burst_id = 0
		}

		// should advance burst_id
		burst_id = int(id/burst_num) + 1
		if _, exist := t.task[conn_id][burst_id]; !exist {
			t.task[conn_id][burst_id] = make(map[uint64]*TaskCall)
		}

		t.task[conn_id][burst_id][id] = new(TaskCall)
	}

	t.conn_num = conn_id
	t.burst_num = int(burst_num)

	return t
}

func main() {
	address := flag.String("address", ":10000", "benchmark server address")
	conn_num := flag.Uint64("conn_num", 1, "benchmark connection number")
	req_num := flag.Uint64("req_num", 1000000, "benchmark request number")
	burst_num := flag.Uint64("burst_num", 1, "benchmark burst number")

	flag.Parse()

	task := NewTask(*req_num, *conn_num, *burst_num)

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
	for conn_id := 1; conn_id <= task.conn_num; conn_id++ {
		conn_wg.Add(1)
		// connection
		go func(r *rpc.Router, conn_wg *sync.WaitGroup, task *Task, conn_id int) {
			conn_task := task.task[conn_id]
			for burst_id := 1; burst_id <= len(conn_task); burst_id++ {
				burst_task := conn_task[burst_id]
				var burst_wg sync.WaitGroup
				for id, _ := range burst_task {
					burst_wg.Add(1)
					req := testpb.NewTestReq()
					req.Id = proto.Uint64(id)
					burst_task[id].start = time.Now()
					burst_task[id].arg = &burst_wg
					r.Call("benchmark", "rpc", req, ClientProcessReponseWaitGroup, burst_task[id], 0)
				}
				burst_wg.Wait()
			}
			conn_wg.Done()
		}(r, &conn_wg, task, conn_id)
	}
	conn_wg.Wait()
	stop := time.Now()

	total_time := stop.Sub(start)
	qps := float64(*req_num*1000*1000*1000) / float64(total_time)

	fmt.Printf("total time: %v total request: %v qps: %v\n",
		total_time, *req_num, qps)

	c := benchmark.NewCollecter(1000 * 1000)
	task.Do(c)

	fmt.Printf("max: %vus\n", c.Max())
	fmt.Printf("min: %vus\n", c.Min())
	fmt.Printf("mean: %vus\n", c.Mean())
	for _, p := range []float64{50.0, 70.0, 90.0, 95.0, 99.0} {
		fmt.Printf("%.1f%% request done: %vus\n", p, c.Percentile(p))
	}
}
