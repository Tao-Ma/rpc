package main

import (
	"benchmark"
	testpb "benchmark/proto_pb_test"
	"flag"
	"fmt"
	proto "github.com/golang/protobuf/proto"
	"os"
	"rpc"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
)

func ClientProcessReponseWaitGroup(p rpc.Payload, arg rpc.RPCCallback_arg, err error) {
	tc := arg.(*TaskCall)
	tc.arg.(chan struct{}) <- struct{}{}

	if err != nil {
		tc.timeout = true
		fmt.Printf("%v\n", err)
		return
	}

	tc.stop = time.Now()
}

type TaskCall struct {
	start time.Time
	stop  time.Time

	arg     interface{}
	timeout bool
}

type Task struct {
	// conn/burst/id
	conn_num  int
	burst_num int
	task      map[int]map[uint64]*TaskCall
}

func (t *Task) Do(c *benchmark.Collector) {
	for _, conn_task := range t.task {
		for _, tc := range conn_task {
			if tc.timeout {
				// TODO: timeout
				continue
			}

			d := tc.stop.Sub(tc.start)
			// us(MicroSecond) uint
			c.Add(uint64(d) / uint64(1000))
		}
	}
}

func NewTask(req_num uint64, conn_num uint64, burst_num uint64) *Task {
	t := new(Task)

	t.task = make(map[int]map[uint64]*TaskCall)

	per_conn_req_num := req_num / conn_num

	conn_id := 0

	for id := uint64(0); id < req_num; id++ {
		// should advance conn_id
		conn_id = int(id/per_conn_req_num) + 1
		if _, exist := t.task[conn_id]; !exist {
			t.task[conn_id] = make(map[uint64]*TaskCall)
		}

		t.task[conn_id][id] = new(TaskCall)
	}

	t.conn_num = conn_id
	t.burst_num = int(burst_num)

	return t
}

var address = flag.String("address", ":10000", "benchmark server address")
var router_per_conn = flag.Bool("router_per_conn", false, "benchmark share Router")
var conn_num = flag.Uint64("conn_num", 1, "benchmark connection number")
var req_num = flag.Uint64("req_num", 1000000, "benchmark request number")
var burst_num = flag.Uint64("burst_num", 1, "benchmark burst number")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()

	// profiling
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	task := NewTask(*req_num, *conn_num, *burst_num)

	hf := rpc.NewRPCHeaderFactory(rpc.NewProtobufFactory())
	_ = hf

	// index 0 is not used
	var routers []*rpc.Router

	for i := 0; i <= task.conn_num; i++ {
		var r *rpc.Router
		var err error

		if *router_per_conn || i == 0 {
			r, err = rpc.NewRouter(nil, nil)
		} else {
			r, err = routers[0], nil
		}
		if err != nil {
			fmt.Println(err)
			return
		} else {
			r.Run()
			defer r.Stop()
		}

		routers = append(routers, r)
	}

	ep_name := "benchmark-"
	for i := 1; i <= task.conn_num; i++ {
		if err := routers[i].Dial(ep_name+strconv.Itoa(i), "tcp", *address, hf); err != nil {
			fmt.Println(err)
			return
		}
	}

	var conn_wg sync.WaitGroup

	start := time.Now()
	for conn_id := 1; conn_id <= task.conn_num; conn_id++ {
		conn_wg.Add(1)
		// connection
		go func(r *rpc.Router, conn_wg *sync.WaitGroup, task *Task, conn_id int) {
			name := ep_name + strconv.Itoa(conn_id)
			conn_task := task.task[conn_id]

			ch := make(chan struct{}, task.burst_num)
			for i := 0; i < task.burst_num; i++ {
				ch <- struct{}{}
			}

			for id, _ := range conn_task {
				<-ch
				req := testpb.NewTestReq()
				req.Id = proto.Uint64(id)

				conn_task[id].start = time.Now()
				conn_task[id].arg = ch
				r.Call(name, "rpc", req, ClientProcessReponseWaitGroup, conn_task[id], 1)
			}
			for i := 0; i < task.burst_num; i++ {
				<-ch
			}
			close(ch)
			conn_wg.Done()
		}(routers[conn_id], &conn_wg, task, conn_id)
	}
	conn_wg.Wait()
	stop := time.Now()

	total_time := stop.Sub(start)
	qps := float64(*req_num*1000*1000*1000) / float64(total_time)

	fmt.Printf("total time: %v total request: %v qps: %.2f\n",
		total_time, *req_num, qps)

	c := benchmark.NewCollecter(1000 * 1000)
	task.Do(c)

	fmt.Printf("max: %vus ", c.Max())
	fmt.Printf("min: %vus ", c.Min())
	fmt.Printf("avg: %.2fus ", c.Mean())
	fmt.Println()

	a := []float64{50.0, 70.0, 90.0, 95.0, 99.0}
	for _, p := range a {
		fmt.Printf("%7.1f%% ", p)
	}
	fmt.Println()
	for _, p := range a {
		fmt.Printf("%6dus ", c.Percentile(p))
	}
	fmt.Println()
}
