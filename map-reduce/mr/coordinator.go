package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type taskState int

const (
	Idle taskState = iota
	InProgress
	Done
)

type MapTask struct {
	filename string
	state    taskState
	startTS  time.Time
}

type ReduceTask struct {
	state   taskState
	startTS time.Time
}

type Coordinator struct {
	mapTasks    []MapTask
	reduceTasks []ReduceTask
	nReduce     int
	done        bool
	mu          sync.Mutex
}

func (c *Coordinator) allMapDone() (done bool) {
	for _, mt := range c.mapTasks {
		if mt.state != Done {
			done = false
			return done
		}
	}
	done = true
	return done
}

func (c *Coordinator) allReduceDone() (done bool) {
	for _, mt := range c.reduceTasks {
		if mt.state != Done {
			done = false
			return done
		}
	}
	done = true
	return done
}

func (c *Coordinator) RequestTask(args *TaskRequest, reply *TaskReply) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, mapTask := range c.mapTasks {
		if mapTask.state == Idle {
			c.mapTasks[i].state, c.mapTasks[i].startTS = InProgress, time.Now()
			*reply = TaskReply{
				Type:     TaskMap,
				TaskID:   i,
				Filename: c.mapTasks[i].filename,
				NReduce:  c.nReduce,
				NMap:     len(c.mapTasks),
			}
			log.Printf("RequestTask: giving %v to worker\n", reply)
			return err
		}
	}

	if c.allMapDone() {
		for i, rTask := range c.reduceTasks {
			if rTask.state == Idle {
				c.reduceTasks[i].state, c.reduceTasks[i].startTS = InProgress, time.Now()
				*reply = TaskReply{
					Type:    TaskReduce,
					TaskID:  i,
					NReduce: c.nReduce,
					NMap:    len(c.mapTasks),
				}
				log.Printf("RequestTask: giving REDUCE %v to worker\n", reply)
				return err
			}
		}

	}

	if c.allMapDone() && c.allReduceDone() {
		c.done = true
		reply.Type = TaskExit
		return err
	}

	reply.Type = TaskNone
	return err
}

func (c *Coordinator) ReportDone(args *TaskCompleteArgs, reply *TaskCompleteReply) error {
	log.Printf("ReportDone: %v %d\n", args.Type, args.TaskID)
	c.mu.Lock()
	defer c.mu.Unlock()

	switch args.Type {
	case TaskMap:
		c.mapTasks[args.TaskID].state = Done

	case TaskReduce:
		c.reduceTasks[args.TaskID].state = Done
	}

	return nil
}

const timeout = 10 * time.Second

func (c *Coordinator) monitor() {
    for {
        time.Sleep(time.Second)
        c.mu.Lock()

        // re-issue timed-out tasks
        now := time.Now()
        for i := range c.mapTasks {
            if c.mapTasks[i].state == InProgress &&
               now.Sub(c.mapTasks[i].startTS) > timeout {
                c.mapTasks[i].state = Idle
            }
        }
        if c.allMapDone() {
            for i := range c.reduceTasks {
                if c.reduceTasks[i].state == InProgress &&
                   now.Sub(c.reduceTasks[i].startTS) > timeout {
                    c.reduceTasks[i].state = Idle
                }
            }
        }

        // declare victory
        if !c.done && c.allMapDone() && c.allReduceDone() {
            c.done = true
        }
        c.mu.Unlock()
    }
}


// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	log.Printf("Coordinator listening on %s\n", sockname)
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{nReduce: nReduce}

	for _, f := range files {
		c.mapTasks = append(c.mapTasks, MapTask{filename: f, state: Idle})
	}
	c.reduceTasks = make([]ReduceTask, nReduce)
	c.server()

	go c.monitor()
	return &c
}
