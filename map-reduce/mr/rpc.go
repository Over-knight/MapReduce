package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type TaskType int

const (
	TaskNone TaskType = iota
	TaskMap
	TaskReduce
	TaskExit
)

type TaskRequest struct{}

type TaskReply struct {
	Type     TaskType
	NReduce  int
	NMap     int
	TaskID   int
	Filename string
}

type TaskCompleteArgs struct {
	Type   TaskType
	TaskID int
}
type TaskCompleteReply struct{}

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
