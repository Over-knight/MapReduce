# Distributed MapReduce in Go

This repository contains a high-performance, distributed MapReduce system implemented in Go. The architecture is designed for scalability and fault tolerance, featuring a centralized coordinator that manages workloads across multiple worker processes. This project is inspired by the Google MapReduce paper and the MIT 6.5840 (Distributed Systems) course.

## System Architecture

The system is composed of three primary layers:

### 1. The Coordinator (`mr/coordinator.go`)
The master process responsible for:
* **Task Scheduling**: Tracking the state of Map and Reduce tasks as Idle, InProgress, or Done.
* **Resource Allocation**: Assigning tasks to workers via RPC requests through the `RequestTask` handler.
* **Fault Tolerance**: A background `monitor` thread detects worker failures by re-issuing tasks that have been in progress for more than 10 seconds.

### 2. The Worker (`mr/worker.go`)
Independent processes that perform the actual computation:
* **Map Phase**: Processes input files, applies the `Map` function, and partitions the results into intermediate JSON-encoded files named `mr-X-Y`.
* **Reduce Phase**: Aggregates intermediate files based on partition IDs, sorts the data, and applies the `Reduce` function to produce the final output files named `mr-out-Y`.

### 3. RPC Layer (`mr/rpc.go`)
Defines the communication protocol used for task requests, status reporting, and data exchange between the coordinator and workers.

---

## Repository Structure

* **`mr/`**: Core logic for the coordinator, worker, and RPC definitions.
* **`mrapps/`**: Implementation of various MapReduce applications:
    * `wc.go`: Word counting logic.
    * `indexer.go`: Distributed inverted indexer.
    * `crash.go`: A test application used to verify fault tolerance by simulating random crashes and delays.
* **`mrcoordinator.go`**: Entry point for starting the coordinator process.
* **`mrworker.go`**: Entry point for starting worker processes with specific logic plugins.

---

## Usage Instructions

### 1. Build an Application Plugin
Compile your desired MapReduce logic into a Go plugin:
go build -buildmode=plugin mrapps/wc.go

### 2. Launch the Coordinator
Initialize the coordinator with your input dataset (e.g., text files):
go run mrcoordinator.go pg-*.txt

### 3. Start Workers
In separate terminal instances, start as many workers as needed:
go run mrworker.go wc.so

Once all tasks are completed, the coordinator will signal the workers to exit, and the results will be available in mr-out-* files.
