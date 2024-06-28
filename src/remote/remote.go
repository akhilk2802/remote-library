// support for generic Remote Object services over sockets
// including a socket wrapper that can drop and/or delay messages arbitrarily
// works with any* objects that can be gob-encoded for serialization
//
// the LeakySocket wrapper for net.Conn is provided in its entirety, and should
// not be changed, though you may extend it with additional helper functions as
// desired.  it is used directly by the test code.
//
// the RemoteObjectError type is also provided in its entirety, and should not
// be changed.
//
// suggested RequestMsg and ReplyMsg types are included to get you started,
// but they are only used internally to the remote library, so you can use
// something else if you prefer
//
// the Service type represents the callee that manages remote objects, invokes
// calls from callers, and returns suitable results and/or remote errors
//
// the StubFactory converts a struct of function declarations into a functional
// caller stub by automatically populating the function definitions.
//
// USAGE:
// the desired usage of this library is as follows (not showing all error-checking
// for clarity and brevity):
//
//	example ServiceInterface known to both client and server, defined as
//	type ServiceInterface struct {
//	    ExampleMethod func(int, int) (int, remote.RemoteObjectError)
//	}
//
//	1. server-side program calls NewService with interface and connection details, e.g.,
//	   obj := &ServiceObject{}
//	   srvc, err := remote.NewService(&ServiceInterface{}, obj, 9999, true, true)
//
//	2. client-side program calls StubFactory, e.g.,
//	   stub := &ServiceInterface{}
//	   err := StubFactory(stub, 9999, true, true)
//
//	3. client makes calls, e.g.,
//	   n, roe := stub.ExampleMethod(7, 14736)
//
// TODO *** here's what needs to be done for Lab 1:
//
//  1. create the Service type and supporting functions, including but not
//     limited to: NewService, Start, Stop, IsRunning, and GetCount (see below)
//
//  2. create the StubFactory which uses reflection to transparently define each
//     method call in the client-side stub (see below)
package remote

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"reflect"
	"sync"
	"time"
)

// LeakySocket
//
// LeakySocket is a wrapper for a net.Conn connection that emulates
// transmission delays and random packet loss. it has its own send
// and receive functions that together mimic an unreliable connection
// that can be customized to stress-test remote service interactions.
type LeakySocket struct {
	s         net.Conn
	isLossy   bool
	lossRate  float32
	msTimeout int
	usTimeout int
	isDelayed bool
	msDelay   int
	usDelay   int
}

// builder for a LeakySocket given a normal socket and indicators
// of whether the connection should experience loss and delay.
// uses default loss and delay values that can be changed using setters.
func NewLeakySocket(conn net.Conn, lossy bool, delayed bool) *LeakySocket {
	ls := &LeakySocket{}
	ls.s = conn
	ls.isLossy = lossy
	ls.isDelayed = delayed
	ls.msDelay = 2
	ls.usDelay = 0
	ls.msTimeout = 15
	ls.usTimeout = 0
	ls.lossRate = 0.05

	return ls
}

// send a byte-string over the socket mimicking unreliability.
// delay is emulated using time.Sleep, packet loss is emulated using RNG
// coupled with time.Sleep to emulate a timeout
func (ls *LeakySocket) SendObject(obj []byte) (bool, error) {
	if obj == nil {
		fmt.Println("SendObject: obj is nil, returning true")
		return true, nil
	}

	if ls.s != nil {
		rand.Seed(time.Now().UnixNano())
		if ls.isLossy && rand.Float32() < ls.lossRate {
			fmt.Println("SendObject: Simulating packet loss")
			time.Sleep(time.Duration(ls.msTimeout)*time.Millisecond + time.Duration(ls.usTimeout)*time.Microsecond)
			return false, nil
		} else {
			if ls.isDelayed {
				fmt.Println("SendObject: Simulating delay")
				time.Sleep(time.Duration(ls.msDelay)*time.Millisecond + time.Duration(ls.usDelay)*time.Microsecond)
			}
			fmt.Println("SendObject: Sending data")
			n, err := ls.s.Write(obj)
			if err != nil {
				fmt.Println("SendObject Write error:", err)
				return false, errors.New("SendObject Write error: " + err.Error())
			}
			fmt.Printf("SendObject: Sent %d bytes\n", n)
			return true, nil
		}
	}
	fmt.Println("SendObject failed, nil socket")
	return false, errors.New("SendObject failed, nil socket")
}

// receive a byte-string over the socket connection.
// no significant change to normal socket receive.
func (ls *LeakySocket) RecvObject() ([]byte, error) {
	fmt.Println("Inside the for loop, in RecvObject")
	if ls.s != nil {
		buf := make([]byte, 4096)
		// var buf []byte
		n := 0
		var err error
		for n <= 0 {
			ls.s.SetReadDeadline(time.Now().Add(10 * time.Second))
			fmt.Println("Waiting to read from socket")
			fmt.Println("Printing conn : ", ls.s)
			n, err = ls.s.Read(buf)
			fmt.Println("value of n:", n)
			if n > 0 {
				fmt.Printf("Read %d bytes\n", n)
				fmt.Println("Priniting from RecvObject :", buf[:n])
				// return nil, nil
				return buf[:n], nil
			}
			if err != nil {
				if err != io.EOF {
					fmt.Println("RecvObject Read error:", err)
					return nil, errors.New("RecvObject Read error: " + err.Error())
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("Read timeout")
					return nil, errors.New("RecvObject Read timeout")
				}
				fmt.Println("RecvObject Read error:", err)
				return nil, errors.New("RecvObject Read error: " + err.Error())
			}
		}
	}
	return nil, errors.New("RecvObject failed, nil socket")
}

// enable/disable emulated transmission delay and/or change the delay parameter
func (ls *LeakySocket) SetDelay(delayed bool, ms int, us int) {
	ls.isDelayed = delayed
	ls.msDelay = ms
	ls.usDelay = us
}

// change the emulated timeout period used with packet loss
func (ls *LeakySocket) SetTimeout(ms int, us int) {
	ls.msTimeout = ms
	ls.usTimeout = us
}

// enable/disable emulated packet loss and/or change the loss rate
func (ls *LeakySocket) SetLossRate(lossy bool, rate float32) {
	ls.isLossy = lossy
	ls.lossRate = rate
}

// close the socket (can also be done on original net.Conn passed to builder)
func (ls *LeakySocket) Close() error {
	return ls.s.Close()
}

// RemoteObjectError
//
// RemoteObjectError is a custom error type used for this library to identify remote methods.
// it is used by both caller and callee endpoints.
type RemoteObjectError struct {
	Err string
}

// getter for the error message included inside the custom error type
func (e *RemoteObjectError) Error() string { return e.Err }

// RequestMsg (this is only a suggestion, can be changed)
//
// RequestMsg represents the request message sent from caller to callee.
// it is used by both endpoints, and uses the reflect package to carry
// arbitrary argument types across the network.
type RequestMsg struct {
	Method string
	Args   []reflect.Value
}

// ReplyMsg (this is only a suggestion, can be changed)
//
// ReplyMsg represents the reply message sent from callee back to caller
// in response to a RequestMsg. it similarly uses reflection to carry
// arbitrary return types along with a success indicator to tell the caller
// whether the call was correctly handled by the callee. also includes
// a RemoteObjectError to specify details of any encountered failure.
type ReplyMsg struct {
	Success bool
	Reply   []reflect.Value
	Err     RemoteObjectError
}

// Service -- server side stub/skeleton
//
// A Service encapsulates a multithreaded TCP server that manages a single
// remote object on a single TCP port, which is a simplification to ease management
// of remote objects and interaction with callers.  Each Service is built
// around a single struct of function declarations. All remote calls are
// handled synchronously, meaning the lifetime of a connection is that of a
// sinngle method call.  A Service can encounter a number of different issues,
// and most of them will result in sending a failure response to the caller,
// including a RemoteObjectError with suitable details.
type Service struct {
	srvIfcType   reflect.Type
	srvIfcValue  reflect.Value
	srvObjValue  reflect.Value
	isRunning    bool
	server       net.Listener
	lossy        bool
	port         int
	delayed      bool
	runningMutex sync.Mutex
	shutdown     chan bool

	// TODO: populate with needed contents including, but not limited to:
	//       - reflect.Type of the Service's interface (struct of Fields)
	//       - reflect.Value of the Service's interface
	//       - reflect.Value of the Service's remote object instance
	//       - status and configuration parameters, as needed
}

// build a new Service instance around a given struct of supported functions,
// a local instance of a corresponding object that supports these functions,
// and arguments to support creation and use of LeakySocket-wrapped connections.
// performs the following:
// -- returns a local error if function struct or object is nil
// -- returns a local error if any function in the struct is not a remote function
// -- if neither error, creates and populates a Service and returns a pointer
func NewService(ifc interface{}, sobj interface{}, port int, lossy bool, delayed bool) (*Service, error) {

	if ifc == nil {
		return nil, errors.New("cannot be nil")
	} else if sobj == nil {
		return nil, errors.New("cannot be nil")
	}

	ifcType := reflect.TypeOf(ifc).Elem()
	fmt.Println("ifcType : ", ifcType)
	for i := 0; i < ifcType.NumField(); i++ {
		methodType := ifcType.Field(i).Type

		hasRemoteObjectError := false
		for j := 0; j < methodType.NumOut(); j++ {
			returnType := methodType.Out(j)
			if returnType.Kind() == reflect.Struct && returnType.Name() == "RemoteObjectError" {
				hasRemoteObjectError = true
				break
			}
		}
		if !hasRemoteObjectError {
			return nil, errors.New("doesn't not return RemotObjectError")
		}
	}

	s := &Service{
		srvIfcType:  reflect.TypeOf(ifc),
		srvIfcValue: reflect.ValueOf(ifc),
		srvObjValue: reflect.ValueOf(sobj),
		lossy:       lossy,
		delayed:     delayed,
		port:        port,
	}
	return s, nil
}

// start the Service's tcp listening connection, update the Service
// status, and start receiving caller connections
func (serv *Service) Start() error {

	serv.runningMutex.Lock()
	defer serv.runningMutex.Unlock()

	if serv.isRunning {
		fmt.Println("Session already running, ignore Start() call.")
		return nil
	}

	var err error
	serv.server, err = net.Listen("tcp", fmt.Sprintf(":%d", serv.port))
	if err != nil {
		return err
	}
	serv.isRunning = true

	serv.shutdown = make(chan bool)
	go func() {
		// Wait for shutdown signal
		<-serv.shutdown
		serv.server.Close()
		serv.isRunning = false
	}()

	// TODO: attempt to start a Service created using NewService
	//
	// if called on a service that is already running, print a warning
	// but don't return an error or do anything else
	//
	// otherwise, start the multithreaded tcp server at the given address
	// and update Service state
	//
	// IMPORTANT: Start() should not be a blocking call. once the Service
	// is started, it should return
	//
	//
	// After the Service is started (not to be done inside of this Start
	//      function, but wherever you want):
	//
	// - accept new connections from client callers until someone calls
	//   Stop on this Service, spawning a thread to handle each one
	//
	// - within each client thread, wrap the net.Conn with a LeakySocket
	//   e.g., if Service accepts a client connection `c`, create a new
	//   LeakySocket ls as `ls := LeakySocket(c, ...)`.  then:
	//
	// 1. receive a byte-string on `ls` using `ls.RecvObject()`
	//
	// 2. decoding the byte-string
	//
	// 3. check to see if the service interface's Type includes a method
	//    with the given name
	//
	// 4. invoke method
	//
	// 5. encode the reply message into a byte-string
	//
	// 6. send the byte-string using `ls.SendObject`, noting that the configuration
	//    of the LossySocket does not guarantee that this will work...
	go listenConnections(serv, serv.shutdown)
	return nil
}

func listenConnections(serv *Service, shutdown <-chan bool) {
	for {
		conn, err := serv.server.Accept()
		if err != nil {
			select {
			case <-shutdown:
				// If shutdown signal is received, break the loop and stop accepting connections
				return
			default:
				fmt.Println("Error accepting connection:", err)
				continue
			}
		}
		go handleConnection(serv, conn)
	}
}

func handleConnection(serv *Service, conn net.Conn) {
	defer conn.Close()

	ls := NewLeakySocket(conn, serv.lossy, serv.delayed)

	for {

		// data -> []byte
		data, err := ls.RecvObject()
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Client closed the connection")
			} else {
				fmt.Println("Error receiving data:", err)
			}
			return
		}

		// Decode data, find method, invoke method, encode reply, send reply
		//handleData(*Service, []bytes)
		response := handleData(serv, data)
		fmt.Println("response : ", response)
		sent, err := ls.SendObject(response)
		if !sent {
			fmt.Println("Error sending data:", err)
			return
		}
	}
}

func decodeGob(data []byte, value interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(value)
}

func encodeError(err error) []byte {
	reply := ReplyMsg{
		Success: false,
		Err:     RemoteObjectError{Err: err.Error()},
	}
	data, _ := encodeGob(reply) // Ignoring error here for simplicity
	return data
}

func encodeGob(value interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// func encodeResult(result []reflect.Value) []byte {
// 	resp := ReplyMsg{Reply: result}
// 	data, err := encodeGob(resp)
// 	if err != nil {
// 		return encodeError(err)
// 	}
// 	return data
// }

func handleData(serv *Service, data []byte) []byte {
	// var req RequestMsg
	// err := decodeGob(data, &req)
	// if err != nil {
	// 	fmt.Println("Error decoding data:", err)
	// 	return encodeError(err)
	// }
	// method := reflect.ValueOf(serv.srvIfcType).MethodByName(req.Method)
	// if !method.IsValid() {
	// 	return encodeError(fmt.Errorf("method %s not found", req.Method))
	// }

	// results := method.Call(req.Args)
	// if len(results) == 0 {
	// 	return encodeError(fmt.Errorf("method %s returned no results", req.Method))
	// }

	// var reply ReplyMsg
	// if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
	// 	reply = ReplyMsg{
	// 		Success: false,
	// 		Err:     RemoteObjectError{err.Error()},
	// 	}
	// } else {
	// 	reply = ReplyMsg{
	// 		Success: true,
	// 		Reply:   results,
	// 	}
	// }

	// replyData, err := encodeGob(reply)
	// if err != nil {
	// 	return encodeError(err)
	// }

	// return replyData

	return data
}

func (serv *Service) GetCount() int {
	// TODO: return the total number of remote calls served successfully by this Service
	return 0
}

func (serv *Service) IsRunning() bool {
	// TODO: return a boolean value indicating whether the Service is running
	return serv.isRunning
}

func (serv *Service) Stop() {

	serv.runningMutex.Lock()
	defer serv.runningMutex.Unlock()

	if !serv.isRunning {
		return
	}

	serv.isRunning = false
	close(serv.shutdown)
	// TODO: stop the Service, change state accordingly, clean up any resources
}

func deserialize(decoder *gob.Decoder, elemName reflect.Value) error {
	// 1. Decode the actual data into the element
	err := decoder.Decode(elemName.Addr().Interface())
	if err != nil {
		return fmt.Errorf("error decoding element value: %w", err)
	}
	// 2. Check if the return type is a struct named "RemoteObjectError"
	if elemName.Kind() == reflect.Struct && elemName.Type().Name() == "RemoteObjectError" {
		// Handle potential error returned from the remote method
		return nil // or handle the error based on your needs (e.g., return the error)
	}
	// 3. No error or special case, return nil
	return nil
}

// StubFactory -- make a client-side stub
//
// StubFactory uses reflection to populate the interface functions to create the
// caller's stub interface. Only works if all functions are exported/public.
// Once created, the interface masks remote calls to a Service that hosts the
// object instance that the functions are invoked on.  The network address of the
// remote Service must be provided with the stub is created, and it may not change later.
// A call to StubFactory requires the following inputs:
// -- a struct of function declarations to act as the stub's interface/proxy
// -- the remote address of the Service as "<ip-address>:<port-number>"
// -- indicator of whether caller-to-callee channel has emulated packet loss
// -- indicator of whether caller-to-callee channel has emulated propagation delay
// performs the following:
// -- returns a local error if function struct is nil
// -- returns a local error if any function in the struct is not a remote function
// -- otherwise, uses relection to access the functions in the given struct and
//
//	populate their function definitions with the required stub functionality
func StubFactory(ifc interface{}, adr string, lossy bool, delayed bool) error {
	if ifc == nil {
		return errors.New("ifc cannot be nil")
	}

	ifcValue := reflect.ValueOf(ifc).Elem()
	ifcType := reflect.TypeOf(ifc).Elem()

	for i := 0; i < ifcType.NumField(); i++ {
		methodType := ifcType.Field(i).Type

		hasRemoteObjectError := false
		for j := 0; j < methodType.NumOut(); j++ {
			returnType := methodType.Out(j)
			if returnType.Kind() == reflect.Struct && returnType.Name() == "RemoteObjectError" {
				hasRemoteObjectError = true
				break
			}
		}
		if !hasRemoteObjectError {
			return errors.New("doesn't return RemoteObjectError")
		}
	}

	for i := 0; i < ifcType.NumField(); i++ {
		field := ifcType.Field(i)
		methodName := field.Name
		fmt.Println("methodName : ", methodName)

		methodFunc := func(args []reflect.Value) (results []reflect.Value) {
			conn, err := net.Dial("tcp", adr)
			if err != nil {
				fmt.Println("Error dialing:", err)

				numOut := field.Type.NumOut()

				var errArray []reflect.Value
				for i := 0; i < numOut-1; i++ {
					errArray = append(errArray, reflect.Zero(field.Type.Out(i)))
				}
				remoteError := RemoteObjectError{Err: "Connection failed"}
				errArray = append(errArray, reflect.ValueOf(remoteError))
				return errArray
			}

			ls := NewLeakySocket(conn, lossy, delayed)

			var buffer bytes.Buffer
			encoder := gob.NewEncoder(&buffer)

			methodN := reflect.ValueOf(field.Name)
			if err := encoder.EncodeValue(methodN); err != nil {
				fmt.Println("Error encoding method name:", err)
			}
			count := 0
			for _, arg := range args {
				count += 1
				if err := encoder.EncodeValue(arg); err != nil {
					fmt.Println("Error encoding argument", count, ":", err)
				}
			}
			if err := encoder.Encode(count); err != nil {
				fmt.Println("Error encoding count:", err)
			}
			fmt.Println("Here, there might be a bug")
			fmt.Println("Buffer bytes : ", buffer.Bytes())
			for {
				sent, err := ls.SendObject(buffer.Bytes())
				if err != nil {
					return
				}
				if !sent {
					fmt.Println("Error while sending the encoded message")
					continue
				} else {
					fmt.Println("Sucessfully sent ")
					break
				}
			}
			fmt.Println("Recieving data from socket")
			var data []byte
			var err1 error
			for {
				fmt.Println("Inside for loop")
				data, err1 = ls.RecvObject()
				fmt.Println("data : ", data)
				fmt.Println("Error inside For loop : ", err1)
				if err1 != nil {
					fmt.Println("Error recieving data : ", err1)
					continue
				}
				if data == nil {
					fmt.Println("No data recieved")
					continue
				} else {
					break
				}
			}

			buf := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(buf)
			numOut := field.Type.NumOut()
			var resultArray = make([]reflect.Value, numOut)
			for i := 0; i < numOut; i++ {
				currentReturnType := field.Type.Out(i)
				var elemName reflect.Value = reflect.New(currentReturnType).Elem()
				err := deserialize(decoder, elemName)
				if err != nil {
					fmt.Errorf("failed to deserialize args name: %w", err)
				}
				resultArray[i] = elemName

			}
			return resultArray
		}
		funcValue := reflect.MakeFunc(field.Type, methodFunc)
		ifcValue.Field(i).Set(funcValue)
	}
	return nil
}
