package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
	// "6.5840/mr"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func Ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// read the content of a file
func ReadFile(filepath string) string {
	contents, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("error reading file contents, %v", err)
	}
	return string(contents)
}

func doMap(tr TaskReply, mapf func(string, string) []KeyValue) {
	fileContents := ReadFile(tr.Filename)

	kva := mapf(tr.Filename, fileContents)

	partitions := make([][]KeyValue, tr.NReduce)

	for _, kv := range kva {
		i := Ihash(kv.Key) % tr.NReduce
		partitions[i] = append(partitions[i], kv)
	}
	for i := 0; i < tr.NReduce; i++ {
		tmp, _ := os.CreateTemp("", "mr-*")
		enc := json.NewEncoder(tmp)

		for _, kv := range partitions[i] {
			_ = enc.Encode(&kv)
		}
		tmp.Close()
		final := fmt.Sprintf("mr-%d-%d", tr.TaskID, i) // mapID-reduceID
		if err := os.Rename(tmp.Name(), final); err != nil {
			log.Fatalf("error in rename of: %v", err)
		}
	}
}

func doReduce(tr TaskReply, reducef func(string, []string) string) {
	// gather intermediate files mr-*-partition
	kvMap := make(map[string][]string)

	for m := 0; m < tr.NMap; m++ {
		filename := fmt.Sprintf("mr-%d-%d",m, tr.TaskID)
		file, err := os.Open(filename)
		if err != nil {
			// Map m might be in progress → retry later
			time.Sleep(500 * time.Millisecond)
			m--
			continue
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue 
			if err := dec.Decode(&kv); err != nil{
				break
			}
			kvMap[kv.Key] = append(kvMap[kv.Key], kv.Value)
		}
		file.Close()
	}

	// sort keys to keep output deterministic
	keys := make([]string, 0, len(kvMap))
	for k := range kvMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tmp, _ := os.CreateTemp("", "mr-out-*")
	for _, k := range keys {
		fmt.Fprintf(tmp, "%v %v\n", k, reducef(k, kvMap[k]))
	}
	tmp.Close()
	os.Rename(tmp.Name(), fmt.Sprintf("mr-out-%d", tr.TaskID))
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	for {
		var reply TaskReply
		ok := call("Coordinator.RequestTask", &TaskRequest{}, &reply)
		if !ok {
			log.Fatal("error getting task")
		}
		// log.Printf("Worker got task: %+v\n", reply)
		switch reply.Type {
		case TaskMap:
			doMap(reply, mapf)
			if !call("Coordinator.ReportDone",
				&TaskCompleteArgs{Type: TaskMap, TaskID: reply.TaskID},
				&TaskCompleteReply{}){
					log.Fatal("report done failed!!!")
				}

		case TaskReduce:
			doReduce(reply, reducef)
			call("Coordinator.ReportDone",
				&TaskCompleteArgs{Type: TaskReduce, TaskID: reply.TaskID},
				&TaskCompleteReply{})
		case TaskNone:
			time.Sleep(1e6)
		case TaskExit:
			return
		}
	}

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
		return false
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
