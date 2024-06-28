# Remote Library based on RPC

### Overview
This repository contains an implementation of a Remote Procedure Call (RPC) library in Go. The library allows you to create and manage a TCP-based RPC server that dynamically invokes methods on a service interface using reflection. The communication between the client and server is handled using Go's encoding/gob package for efficient binary encoding and decoding.

### Features
 - TCP-based server for handling RPC calls.
 - Dynamic method invocation using reflection.
 - Binary encoding and decoding using gob.
 - Graceful handling of client connections.
 - Asynchronous handling of multiple client connections.
 - Concurrency handling
