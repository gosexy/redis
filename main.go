/*
  Copyright (c) 2013 José Carlos Nieto, https://menteslibres.net/xiam

  Permission is hereby granted, free of charge, to any person obtaining
  a copy of this software and associated documentation files (the
  "Software"), to deal in the Software without restriction, including
  without limitation the rights to use, copy, modify, merge, publish,
  distribute, sublicense, and/or sell copies of the Software, and to
  permit persons to whom the Software is furnished to do so, subject to
  the following conditions:

  The above copyright notice and this permission notice shall be
  included in all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
  EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
  MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
  NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
  LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
  OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
  WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

/*
	(Unofficial) bindings for the official C redis client (hiredis).

	Supports the complete set of commands for the 2.6 series.
*/
package redis

/*
#cgo CFLAGS: -I./_hiredis -DCGO=1

#include <stdarg.h>
#include <stdlib.h>

#include "net.h"
#include "sds.h"
#include "hiredis.h"
#include "async.h"

typedef struct redisEvent {

} redisEvent;

// Redis events with goroutines.
typedef struct redisGoroutineEvents {
	redisAsyncContext *context;
} redisGoroutineEvents;

int redisGetReplyType(redisReply *);

struct timeval redisTimeVal(long, long);
void getCallback(redisAsyncContext *, void *, void *);

int redisGetReplyType(redisReply *);
int redisAsyncCommandArgvWrapper(redisAsyncContext *, void *, void *, int, const char **, const size_t *);
redisReply *redisReplyGetElement(redisReply *, int i);

int redisGoroutineAttach(redisAsyncContext *ac, redisEvent *);
void redisGoroutineWriteEvent(void *arg);
void redisGoroutineReadEvent(void *arg);
void redisForceAsyncFree(redisAsyncContext *ac);

*/
import "C"

import (
	"errors"
	"fmt"
	"menteslibres.net/gosexy/to"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

const (
	pollWait = 100
)

// Error messages
var (
	ErrFailedAllocation    = errors.New(`Got memory allocation problem.`)
	ErrNilReply            = errors.New(`Received a nil response.`)
	ErrNotConnected        = errors.New(`Client is not connected.`)
	ErrMissingCommand      = errors.New(`Missing command.`)
	ErrUnexpectedResponse  = errors.New(`Unexpected response from server.`)
	ErrExpectingPairs      = errors.New(`Expecting (field -> value) pairs.`)
	ErrNonBlockingRequired = errors.New(`This command requires a non-blocking connection.`)
	ErrMissingDestination  = errors.New(`Missing destination.`)
	ErrInvalidDestination  = errors.New(`Destination must be a pointer.`)
)

// Avoids creating reflect.TypeOf([]interface{}{}) in setReplyValue.
var interfaceSliceType = reflect.TypeOf([]interface{}{})

// Workaround for creating a C's struct timeval.
func cStructTimeval(timeout time.Duration) C.struct_timeval {
	return C.redisTimeVal(C.long(int64(timeout/time.Second)), C.long(int64(timeout%time.Millisecond)))
}

// A redis client
type Client struct {
	ctx   *C.redisContext
	async *C.redisAsyncContext
	ev    *C.redisEvent
	loop  bool
}

// File descriptor event map.
var fdev map[int]map[int]func()

func init() {

	fdev = map[int]map[int]func(){
		'r': make(map[int]func()),
		'w': make(map[int]func()),
	}

	startServer()
}

// Creates a new redis client.
func New() *Client {
	self := &Client{}
	return self
}

//export redisEventAddWrite
func redisEventAddWrite(evptr unsafe.Pointer) {
	ev := (*C.redisGoroutineEvents)(evptr)
	ctx := (*C.redisContext)(&ev.context.c)
	fd := int(ctx.fd)

	fdev['w'][fd] = func() {
		C.redisGoroutineWriteEvent(unsafe.Pointer(ev))
	}

	pollserver.WaitWrite(fd)
}

//export redisEventDelWrite
func redisEventDelWrite(evptr unsafe.Pointer) {
	ev := (*C.redisGoroutineEvents)(evptr)
	ctx := (*C.redisContext)(&ev.context.c)
	fd := int(ctx.fd)
	fdev['w'][fd] = nil
	delete(fdev['w'], fd)
	pollserver.EvictWrite(fd)

}

//export redisEventDelRead
func redisEventDelRead(evptr unsafe.Pointer) {
	ev := (*C.redisGoroutineEvents)(evptr)
	ctx := (*C.redisContext)(&ev.context.c)
	fd := int(ctx.fd)
	fdev['r'][fd] = nil
	delete(fdev['r'], fd)
	pollserver.EvictRead(fd)
}

//export redisEventCleanup
func redisEventCleanup(evptr unsafe.Pointer) {
	ev := (*C.redisGoroutineEvents)(evptr)
	ctx := (*C.redisContext)(&ev.context.c)
	fd := int(ctx.fd)
	delete(fdev['r'], fd)
	delete(fdev['w'], fd)
	pollserver.EvictRead(fd)
	pollserver.EvictWrite(fd)
}

//export redisEventAddRead
func redisEventAddRead(evptr unsafe.Pointer) {
	ev := (*C.redisGoroutineEvents)(evptr)
	ctx := (*C.redisContext)(&ev.context.c)
	fd := int(ctx.fd)

	fdev['r'][fd] = func() {
		C.redisGoroutineReadEvent(unsafe.Pointer(ev))
	}
	pollserver.WaitRead(fd)
}

//export asyncReceive
func asyncReceive(a unsafe.Pointer, b unsafe.Pointer) {
	if b != nil {
		(*(*func(unsafe.Pointer))(unsafe.Pointer(&b)))(a)
	}
}

// Associates context with client.
func (self *Client) setContext(ctx *C.redisContext) error {

	if ctx == nil {
		return ErrFailedAllocation
	} else if ctx.err > 0 {
		err := errors.New(C.GoString(&ctx.errstr[0]))
		C.redisFree(ctx)
		return err
	}

	self.ctx = ctx

	self.async = nil

	return nil
}

// Creates an asynchronous connection.
func (self *Client) asyncConnect(ctx *C.redisContext) error {

	err := self.setContext(ctx)

	if err != nil {
		return err
	}

	self.async = C.redisAsyncInitialize(self.ctx)

	if self.async == nil {
		C.redisFree(self.ctx)
		return ErrFailedAllocation
	}

	C.__redisAsyncCopyError(self.async)

	C.redisGoroutineAttach(self.async, self.ev)

	return nil
}

// Connects the client to the given host and port.
func (self *Client) Connect(host string, port uint) error {
	var ctx *C.redisContext

	chost := C.CString(host)

	ctx = C.redisConnect(
		chost,
		C.int(port),
	)

	C.free(unsafe.Pointer(chost))

	return self.setContext(ctx)
}

// Creates a non-blocking connection with the given host and port.
func (self *Client) ConnectNonBlock(host string, port uint) error {

	var ctx *C.redisContext

	chost := C.CString(host)

	ctx = C.redisConnectNonBlock(
		chost,
		C.int(port),
	)

	C.free(unsafe.Pointer(chost))

	return self.asyncConnect(ctx)
}

// Connects to the given host and port, giving up after timeout.
func (self *Client) ConnectWithTimeout(host string, port uint, timeout time.Duration) error {

	var ctx *C.redisContext

	chost := C.CString(host)

	ctx = C.redisConnectWithTimeout(
		chost,
		C.int(port),
		cStructTimeval(timeout),
	)

	C.free(unsafe.Pointer(chost))

	return self.setContext(ctx)
}

// Creates a non-blocking connection between the client and the given UNIX
// socket.
func (self *Client) ConnectUnixNonBlock(path string) error {
	var ctx *C.redisContext

	cpath := C.CString(path)

	ctx = C.redisConnectUnixNonBlock(
		cpath,
	)

	C.free(unsafe.Pointer(cpath))

	return self.asyncConnect(ctx)
}

// Connects the client to the given UNIX socket.
func (self *Client) ConnectUnix(path string) error {
	var ctx *C.redisContext

	cpath := C.CString(path)

	ctx = C.redisConnectUnix(
		cpath,
	)

	C.free(unsafe.Pointer(cpath))

	return self.setContext(ctx)
}

// Connects the client to the given UNIX socket, giving up after timeout.
func (self *Client) ConnectUnixWithTimeout(path string, timeout time.Duration) error {
	var ctx *C.redisContext

	cpath := C.CString(path)

	ctx = C.redisConnectUnixWithTimeout(
		cpath,
		cStructTimeval(timeout),
	)

	C.free(unsafe.Pointer(cpath))

	return self.setContext(ctx)
}

func setReplyValue(v reflect.Value, raw unsafe.Pointer) error {

	reply := (*C.redisReply)(raw)

	switch C.redisGetReplyType(reply) {
	// Received a string.
	case C.REDIS_REPLY_STRING, C.REDIS_REPLY_STATUS, C.REDIS_REPLY_ERROR:
		s := C.GoStringN(reply.str, reply.len)
		// Destination type.
		switch v.Kind() {
		case reflect.String:
			// string -> string
			v.Set(reflect.ValueOf(s))
		case reflect.Int:
			// string -> int
			v.Set(reflect.ValueOf(int(to.Int64(s))))
		case reflect.Int8:
			// string -> int8
			v.Set(reflect.ValueOf(int8(to.Int64(s))))
		case reflect.Int16:
			// string -> int16
			v.Set(reflect.ValueOf(int16(to.Int64(s))))
		case reflect.Int32:
			// string -> int32
			v.Set(reflect.ValueOf(int32(to.Int64(s))))
		case reflect.Int64:
			// string -> int64
			v.Set(reflect.ValueOf(to.Int64(s)))
		case reflect.Uint:
			// string -> uint
			v.Set(reflect.ValueOf(uint(to.Uint64(s))))
		case reflect.Uint8:
			// string -> uint8
			v.Set(reflect.ValueOf(uint8(to.Uint64(s))))
		case reflect.Uint16:
			// string -> uint16
			v.Set(reflect.ValueOf(uint16(to.Uint64(s))))
		case reflect.Uint32:
			// string -> uint32
			v.Set(reflect.ValueOf(uint32(to.Uint64(s))))
		case reflect.Uint64:
			// string -> uint64
			v.Set(reflect.ValueOf(to.Uint64(s)))
		case reflect.Bool:
			// string -> bool
			v.Set(reflect.ValueOf(to.Bool(s)))
		case reflect.Interface:
			v.Set(reflect.ValueOf(s))
		default:
			return fmt.Errorf("Unsupported conversion: redis string to %v", v.Kind())
		}
	// Received integer.
	case C.REDIS_REPLY_INTEGER:
		switch v.Kind() {
		case reflect.String:
			// integer -> string
			v.Set(reflect.ValueOf(to.String(reply.integer)))
		case reflect.Int:
			// Different integer types.
			v.Set(reflect.ValueOf(int(reply.integer)))
		case reflect.Int8:
			v.Set(reflect.ValueOf(int8(reply.integer)))
		case reflect.Int16:
			v.Set(reflect.ValueOf(int16(reply.integer)))
		case reflect.Int32:
			v.Set(reflect.ValueOf(int32(reply.integer)))
		case reflect.Int64:
			v.Set(reflect.ValueOf(int64(reply.integer)))
		case reflect.Uint:
			v.Set(reflect.ValueOf(uint(reply.integer)))
		case reflect.Uint8:
			v.Set(reflect.ValueOf(uint8(reply.integer)))
		case reflect.Uint16:
			v.Set(reflect.ValueOf(uint16(reply.integer)))
		case reflect.Uint32:
			v.Set(reflect.ValueOf(uint32(reply.integer)))
		case reflect.Uint64:
			v.Set(reflect.ValueOf(uint64(reply.integer)))
		case reflect.Bool:
			b := false
			if reply.integer == 1 {
				b = true
			}
			v.Set(reflect.ValueOf(b))
		case reflect.Interface:
			v.Set(reflect.ValueOf(reply.integer))
		default:
			return fmt.Errorf("Unsupported conversion: redis integer (%d) to %v", reply.integer, v.Kind())
		}
	case C.REDIS_REPLY_ARRAY:
		switch v.Kind() {
		case reflect.Slice, reflect.Interface:
			var err error
			var elements reflect.Value
			total := int(reply.elements)
			if v.Kind() == reflect.Interface {
				// interface{} is not an slice, but it can hold one, however you can't just
				// use v.Type() here, since v.Type() is reflect.TypeOf(interface{}{}). We
				// need reflect.Type([]interface{}{}) instead.
				elements = reflect.MakeSlice(interfaceSliceType, total, total)
			} else {
				// Other types must have the right element type.
				elements = reflect.MakeSlice(v.Type(), total, total)
			}
			for i := 0; i < total; i++ {
				item := C.redisReplyGetElement(reply, C.int(i))
				err = setReplyValue(elements.Index(i), unsafe.Pointer(item))
				if err != nil {
					return err
				}
				// Parent's freeReplyObject already takes care.
				// C.freeReplyObject(unsafe.Pointer(item))
			}
			v.Set(elements)
		default:
			return fmt.Errorf("Unsupported conversion: redis array to %v", v.Kind())
		}
	case C.REDIS_REPLY_NIL:
		/*
			Read here: https://github.com/gosexy/redis/issues/12

			'nil' was being considered and invalid response type, but that caused
			trouble within perfectly valid situations, so 'nil' is now an accepted
			response type.
		*/
		v.Set(reflect.Zero(v.Type()))
		//return ErrNilReply
		return nil
	default:
		return fmt.Errorf("Unknown redis reply type: %v", C.redisGetReplyType(reply))
	}

	return nil
}

// Sends a raw command and stores the response into a destination variable.
func (self *Client) Command(dest interface{}, values ...interface{}) error {

	argc := len(values)
	argv := make([][]byte, argc)

	for i := 0; i < argc; i++ {
		argv[i] = to.Bytes(values[i])
	}

	return self.command(dest, argv...)
}

func (self *Client) bcommand(c chan []string, dest interface{}, values ...[]byte) error {

	var i int
	var err error

	if self.ctx == nil {
		return ErrNotConnected
	}

	if len(values) < 1 {
		return ErrMissingCommand
	}

	argc := len(values)
	argv := make([](*C.char), argc)
	argvlen := make([]C.size_t, argc)

	for i, _ = range values {
		argv[i] = C.CString(string(values[i]))
		argvlen[i] = C.size_t(len(values[i]))
	}

	if self.async == nil {

		// Using panic() because we don't really want the user to try subscribing
		// on a blocking conection as it may produce some difficult to catch bugs.
		panic(ErrNonBlockingRequired.Error())
		//return ErrNonBlockingRequired

	} else {

		self.loop = false

		ok := make(chan *C.redisReply)

		var fn = func(r unsafe.Pointer) {
			if self.loop {
				ok <- (*C.redisReply)(unsafe.Pointer(r))
			} else {
				ok <- nil
			}
		}

		var asyncReceiveCallback = asyncReceive
		var ptr = unsafe.Pointer(&asyncReceiveCallback)
		var fnptr = unsafe.Pointer(&fn)

		self.loop = true

		wait := C.redisAsyncCommandArgvWrapper(self.async, (*(*unsafe.Pointer)(ptr)), (*(*unsafe.Pointer)(fnptr)), C.int(argc), &argv[0], (*C.size_t)(&argvlen[0]))

		for i = 0; i < len(argv); i++ {
			C.free(unsafe.Pointer(argv[i]))
		}

		if wait != C.REDIS_OK {
			return ErrUnexpectedResponse
		}

		for self.loop {

			select {

			case res := <-ok:

				if res == nil {
					return ErrUnexpectedResponse
				}

				ptr := (unsafe.Pointer)(res)

				switch C.redisGetReplyType(res) {
				case C.REDIS_REPLY_ERROR:
					return ErrUnexpectedResponse
				}

				if dest == nil {
					return ErrMissingDestination
				}

				rv := reflect.ValueOf(dest)

				if rv.Kind() != reflect.Ptr || rv.IsNil() {
					return ErrInvalidDestination
				}

				err = setReplyValue(rv.Elem(), ptr)

				C.freeReplyObject(ptr)

				if err != nil {
					return err
				}

				c <- reflect.Indirect(rv).Interface().([]string)

			}
		}
	}

	return nil
}

func (self *Client) command(dest interface{}, values ...[]byte) error {

	var i int
	var reply *C.redisReply

	if self.ctx == nil {
		return ErrNotConnected
	}

	if len(values) < 1 {
		return ErrMissingCommand
	}

	command := strings.ToUpper(string(values[0]))

	argc := len(values)
	argv := make([](*C.char), argc)
	argvlen := make([]C.size_t, argc)

	for i, _ = range values {
		argv[i] = C.CString(string(values[i]))
		argvlen[i] = C.size_t(len(values[i]))
	}

	defer func() {
		if reply != nil {
			C.freeReplyObject(unsafe.Pointer(reply))
		}
		for i = 0; i < len(argv); i++ {
			C.free(unsafe.Pointer(argv[i]))
		}
	}()

	if self.async == nil {

		reply = (*C.redisReply)(C.redisCommandArgv(self.ctx, C.int(argc), &argv[0], (*C.size_t)(&argvlen[0])))

	} else {

		ok := make(chan *C.redisReply)

		var fn = func(r unsafe.Pointer) {
			ok <- (*C.redisReply)(unsafe.Pointer(r))
		}

		var asyncReceiveCallback = asyncReceive
		var ptr = unsafe.Pointer(&asyncReceiveCallback)
		var fnptr = unsafe.Pointer(&fn)

		wait := C.redisAsyncCommandArgvWrapper(self.async, (*(*unsafe.Pointer)(ptr)), (*(*unsafe.Pointer)(fnptr)), C.int(argc), &argv[0], (*C.size_t)(&argvlen[0]))

		if wait != C.REDIS_OK {
			return ErrUnexpectedResponse
		}

		if command == "UNSUBSCRIBE" || command == "PUNSUBSCRIBE" {
			return nil
		}

		reply = <-ok
	}

	if reply == nil {
		return ErrNilReply
	}

	if C.redisGetReplyType(reply) == C.REDIS_REPLY_ERROR {
		return errors.New(C.GoString(reply.str))
	}

	if dest == nil {
		return nil
	}

	rv := reflect.ValueOf(dest)

	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrInvalidDestination
	}

	return setReplyValue(rv.Elem(), unsafe.Pointer(reply))
}

/*
	API implementation.
*/

/*
If key already exists and is a string, this command appends the value at the
end of the string. If key does not exist it is created and set as an empty
string, so APPEND will be similar to SET in this special case.

http://redis.io/commands/append
*/
func (self *Client) Append(key string, value interface{}) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("APPEND"),
		[]byte(key),
		to.Bytes(value),
	)
	return ret, err
}

/*
Request for authentication in a password-protected Redis server. Redis can be
instructed to require a password before allowing clients to execute commands.
This is done using the requirepass directive in the configuration file.

http://redis.io/commands/auth
*/
func (self *Client) Auth(password string) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("AUTH"),
		[]byte(password),
	)
	return ret, err
}

/*
Instruct Redis to start an Append Only File rewrite process. The rewrite will
create a small optimized version of the current Append Only File.

http://redis.io/commands/bgwriteaof
*/
func (self *Client) BgRewriteAOF() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("BGREWRITEAOF"),
	)
	return ret, err
}

/*
Save the DB in background. The OK code is immediately returned. Redis forks, the
parent continues to serve the clients, the child saves the DB on disk then exits.

A client my be able to check if the operation succeeded using the LASTSAVE
command.

http://redis.io/commands/bgsave
*/
func (self *Client) BgSave() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("BGSAVE"),
	)
	return ret, err
}

/*
Count the number of set bits (population counting) in a string.

http://redis.io/commands/bitcount
*/
func (self *Client) BitCount(key string, params ...int64) (int64, error) {
	var ret int64
	args := make([][]byte, len(params)+2)
	args[0] = []byte("BITCOUNT")
	args[1] = []byte(key)
	for i, _ := range params {
		args[2+i] = to.Bytes(params[i])
	}
	err := self.command(&ret, args...)
	return ret, err
}

/*
Perform a bitwise operation between multiple keys (containing string values)
and store the result in the destination key.

http://redis.io/commands/bitop
*/
func (self *Client) BitOp(op string, dest string, keys ...string) (int64, error) {
	var ret int64
	args := make([][]byte, len(keys)+3)
	args[0] = []byte("BITOP")
	args[1] = []byte(op)
	args[2] = []byte(dest)
	for i, _ := range keys {
		args[3+i] = to.Bytes(keys[i])
	}
	err := self.command(&ret, args...)
	return ret, err
}

/*
BLPOP is a blocking list pop primitive. It is the blocking version of LPOP
because it blocks the connection when there are no elements to pop from any of
the given lists. An element is popped from the head of the first list that is
non-empty, with the given keys being checked in the order that they are given.

http://redis.io/commands/blpop
*/
func (self *Client) BLPop(timeout uint64, keys ...string) ([]string, error) {
	var ret []string
	args := make([][]byte, len(keys)+2)
	args[0] = []byte("BLPOP")

	i := 0

	for _, key := range keys {
		i += 1
		args[i] = to.Bytes(key)
	}

	args[1+i] = to.Bytes(timeout)

	err := self.command(&ret, args...)
	return ret, err
}

/*
BRPOP is a blocking list pop primitive. It is the blocking version of RPOP
because it blocks the connection when there are no elements to pop from any of
the given lists. An element is popped from the tail of the first list that is
non-empty, with the given keys being checked in the order that they are given.

http://redis.io/commands/brpop
*/
func (self *Client) BRPop(timeout uint64, keys ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(keys)+2)
	args[0] = []byte("BRPOP")

	i := 0

	for _, key := range keys {
		i += 1
		args[i] = to.Bytes(key)
	}
	args[1+i] = to.Bytes(timeout)

	err := self.command(&ret, args...)
	return ret, err
}

/*
BRPOPLPUSH is the blocking variant of RPOPLPUSH. When source contains elements,
this command behaves exactly like RPOPLPUSH. When source is empty, Redis will
block the connection until another client pushes to it or until timeout is
reached. A timeout of zero can be used to block indefinitely.

http://redis.io/commands/brpoplpush
*/
func (self *Client) BRPopLPush(source string, destination string, timeout int64) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("BRPOPLPUSH"),
		[]byte(source),
		[]byte(destination),
		to.Bytes(timeout),
	)
	return ret, err
}

/*
The CLIENT KILL command closes a given client connection identified by ip:port.

http://redis.io/commands/client-kill
*/
func (self *Client) ClientKill(ip string, port uint) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("CLIENT"),
		[]byte("KILL"),
		to.Bytes(fmt.Sprintf("%s:%d", ip, port)),
	)
	return ret, err
}

/*
The CLIENT LIST command returns information and statistics about the client
connections server in a mostly human readable format.

http://redis.io/commands/client-list
*/
func (self *Client) ClientList() ([]string, error) {
	var ret []string
	err := self.command(
		&ret,
		[]byte("CLIENT"),
		[]byte("LIST"),
	)
	return ret, err
}

/*
The CLIENT GETNAME returns the name of the current connection as set by CLIENT
SETNAME. Since every new connection starts without an associated name, if no
name was assigned a null bulk reply is returned.

http://redis.io/commands/client-getname
*/
func (self *Client) ClientGetName() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("CLIENT"),
		[]byte("GETNAME"),
	)
	return ret, err
}

/*
The CLIENT SETNAME command assigns a name to the current connection.

http://redis.io/commands/client-setname
*/
func (self *Client) ClientSetName(connectionName string) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("CLIENT"),
		[]byte("SETNAME"),
		to.Bytes(connectionName),
	)
	return ret, err
}

/*
The CONFIG GET command is used to read the configuration parameters of a running
Redis server. Not all the configuration parameters are supported in Redis 2.4,
while Redis 2.6 can read the whole configuration of a server using this command.

http://redis.io/commands/config-get
*/
func (self *Client) ConfigGet(parameter string) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("CONFIG"),
		[]byte("GET"),
		[]byte(parameter),
	)
	return ret, err
}

/*
The CONFIG SET command is used in order to reconfigure the server at run time
without the need to restart Redis. You can change both trivial parameters or
switch from one to another persistence option using this command.

http://redis.io/commands/config-set
*/
func (self *Client) ConfigSet(parameter string, value interface{}) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("CONFIG"),
		[]byte("SET"),
		[]byte(parameter),
		to.Bytes(value),
	)
	return ret, err
}

/*
Resets the statistics reported by Redis using the INFO command.

http://redis.io/commands/config-resetstat
*/
func (self *Client) ConfigResetStat() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("CONFIG"),
		[]byte("RESETSTAT"),
	)
	return ret, err
}

/*
Return the number of keys in the currently-selected database.

http://redis.io/commands/dbsize
*/
func (self *Client) DbSize() (uint64, error) {
	var ret uint64
	err := self.command(
		&ret,
		[]byte("DBSIZE"),
	)
	return ret, err
}

/*
DEBUG OBJECT is a debugging command that should not be used by clients. Check
the OBJECT command instead.

http://redis.io/commands/debug-object
*/
func (self *Client) DebugObject(key string) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("DEBUG OBJECT"),
		to.Bytes(key),
	)
	return ret, err
}

/*
DEBUG SEGFAULT performs an invalid memory access that crashes Redis. It is used
to simulate bugs during the development.

http://redis.io/commands/debug-segfault
*/
func (self *Client) DebugSegfault() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("DEBUG SEGFAULT"),
	)
	return ret, err
}

/*
Decrements the number stored at key by one. If the key does not exist, it is set
to 0 before performing the operation. An error is returned if the key contains a
value of the wrong type or contains a string that can not be represented as
integer. This operation is limited to 64 bit signed integers.

http://redis.io/commands/decr
*/
func (self *Client) Decr(key string) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("DECR"),
		[]byte(key),
	)
	return ret, err
}

/*
Decrements the number stored at key by decrement. If the key does not exist, it
is set to 0 before performing the operation. An error is returned if the key
contains a value of the wrong type or contains a string that can not be
represented as integer. This operation is limited to 64 bit signed integers.

http://redis.io/commands/decrby
*/
func (self *Client) DecrBy(key string, decrement int64) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("DECRBY"),
		[]byte(key),
		to.Bytes(decrement),
	)
	return ret, err
}

/*
Removes the specified keys. A key is ignored if it does not exist.

http://redis.io/commands/del
*/

func (self *Client) Del(keys ...string) (int64, error) {
	var ret int64
	args := make([][]byte, len(keys)+1)
	args[0] = []byte("DEL")
	for i, key := range keys {
		args[1+i] = to.Bytes(key)
	}
	err := self.command(&ret, args...)
	return ret, err
}

/*
Flushes all previously queued commands in a transaction and restores the
connection state to normal.

http://redis.io/commands/discard
*/
func (self *Client) Discard() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("DISCARD"),
	)
	return ret, err
}

/*
Serialize the value stored at key in a Redis-specific format and return it to
the user. The returned value can be synthesized back into a Redis key using the
RESTORE command.

http://redis.io/commands/dump
*/
func (self *Client) Dump(key string) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("DUMP"),
		[]byte(key),
	)
	return ret, err
}

/*
Returns message.

http://redis.io/commands/echo
*/
func (self *Client) Echo(message interface{}) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("ECHO"),
		to.Bytes(message),
	)
	return ret, err
}

/*
Executes all previously queued commands in a transaction and restores the
connection state to normal.

http://redis.io/commands/exec
*/
func (self *Client) Exec() ([]interface{}, error) {
	var ret []interface{}
	err := self.command(
		&ret,
		[]byte("EXEC"),
	)
	return ret, err
}

/*
Returns if key exists.

http://redis.io/commands/exists
*/
func (self *Client) Exists(key string) (bool, error) {
	var ret bool
	err := self.command(
		&ret,
		[]byte("EXISTS"),
		to.Bytes(key),
	)
	return ret, err
}

/*
Set a timeout on key. After the timeout has expired, the key will automatically
be deleted. A key with an associated timeout is often said to be volatile in
Redis terminology.

http://redis.io/commands/expire
*/
func (self *Client) Expire(key string, seconds uint64) (bool, error) {
	var ret bool
	err := self.command(
		&ret,
		[]byte("EXPIRE"),
		[]byte(key),
		to.Bytes(seconds),
	)
	return ret, err
}

/*
EXPIREAT has the same effect and semantic as EXPIRE, but instead of specifying
the number of seconds representing the TTL (time to live), it takes an absolute
Unix timestamp (seconds since January 1, 1970).

http://redis.io/commands/expireat
*/
func (self *Client) ExpireAt(key string, unixTime uint64) (bool, error) {
	var ret bool
	err := self.command(
		&ret,
		[]byte("EXPIREAT"),
		[]byte(key),
		to.Bytes(unixTime),
	)
	return ret, err
}

/*
Delete all the keys of all the existing databases, not just the currently
selected one. This command never fails.

http://redis.io/commands/flushall
*/
func (self *Client) FlushAll() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("FLUSHALL"),
	)
	return ret, err
}

/*
Delete all the keys of the currently selected DB. This command never fails.

http://redis.io/commands/flushdb
*/
func (self *Client) FlushDB() (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("FLUSHDB"),
	)
	return ret, err
}

/*
Get the value of key. If the key does not exist the special value nil is
returned. An error is returned if the value stored at key is not a string,
because GET only handles string values.

http://redis.io/commands/get
*/
func (self *Client) Get(key string) (string, error) {
	var s string
	err := self.command(&s,
		[]byte("GET"),
		[]byte(key),
	)
	return s, err
}

/*
Returns the bit value at offset in the string value stored at key.

http://redis.io/commands/getbit
*/
func (self *Client) GetBit(key string, offset int64) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("GETBIT"),
		[]byte(key),
		to.Bytes(offset),
	)
	return ret, err
}

/*
Returns the substring of the string value stored at key, determined by the
offsets start and end (both are inclusive). Negative offsets can be used in
order to provide an offset starting from the end of the string. So -1 means
the last character, -2 the penultimate and so forth.

http://redis.io/commands/getrange
*/
func (self *Client) GetRange(key string, start int64, end int64) (string, error) {
	var ret string
	err := self.command(&ret,
		[]byte("GETRANGE"),
		[]byte(key),
		to.Bytes(start),
		to.Bytes(end),
	)
	return ret, err
}

/*
Atomically sets key to value and returns the old value stored at key. Returns an
error when key exists but does not hold a string value.

http://redis.io/commands/getset
*/
func (self *Client) GetSet(key string, value interface{}) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte(string("GETSET")),
		to.Bytes(key),
		to.Bytes(value),
	)
	return ret, err
}

/*
Removes the specified fields from the hash stored at key. Specified fields that
do not exist within this hash are ignored. If key does not exist, it is treated
as an empty hash and this command returns 0.

http://redis.io/commands/hdel
*/
func (self *Client) HDel(key string, fields ...string) (int64, error) {
	var ret int64

	args := make([][]byte, len(fields)+2)
	args[0] = []byte("HDEL")
	args[1] = []byte(key)

	for i, _ := range fields {
		args[2+i] = to.Bytes(fields[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns if field is an existing field in the hash stored at key.

http://redis.io/commands/hexists
*/
func (self *Client) HExists(key string, field string) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("HEXISTS"),
		[]byte(key),
		[]byte(field),
	)

	return ret, err
}

/*
Returns the value associated with field in the hash stored at key.

http://redis.io/commands/hget
*/
func (self *Client) HGet(key string, field string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("HGET"),
		[]byte(key),
		[]byte(field),
	)

	return ret, err
}

/*
Returns all fields and values of the hash stored at key. In the returned value,
every field name is followed by its value, so the length of the reply is twice
the size of the hash.

http://redis.io/commands/hgetall
*/
func (self *Client) HGetAll(key string) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("HGETALL"),
		[]byte(key),
	)

	return ret, err
}

/*
Increments the number stored at field in the hash stored at key by increment. If
key does not exist, a new key holding a hash is created. If field does not exist
the value is set to 0 before the operation is performed.

http://redis.io/commands/hincrby
*/
func (self *Client) HIncrBy(key string, field string, increment int64) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("HINCRBY"),
		[]byte(key),
		[]byte(field),
		to.Bytes(increment),
	)

	return ret, err
}

/*
Increment the specified field of an hash stored at key, and representing a
floating point number, by the specified increment. If the field does not exist,
it is set to 0 before performing the operation.

http://redis.io/commands/hincrbyfloat
*/
func (self *Client) HIncrByFloat(key string, field string, increment float64) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("HINCRBYFLOAT"),
		[]byte(key),
		[]byte(field),
		to.Bytes(increment),
	)

	return ret, err
}

/*
Returns all field names in the hash stored at key.

http://redis.io/commands/hkeys
*/
func (self *Client) HKeys(key string) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("HKEYS"),
		[]byte(key),
	)

	return ret, err

}

/*
Returns the number of fields contained in the hash stored at key.

http://redis.io/commands/hlen
*/
func (self *Client) HLen(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("HLEN"),
		[]byte(key),
	)

	return ret, err
}

/*
Returns the values associated with the specified fields in the hash stored at key.

http://redis.io/commands/hmget
*/
func (self *Client) HMGet(key string, fields ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(fields)+2)
	args[0] = []byte("HMGET")
	args[1] = []byte(key)

	for i, field := range fields {
		args[2+i] = to.Bytes(field)
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Sets the specified fields to their respective values in the hash stored at key.
This command overwrites any existing fields in the hash. If key does not exist,
a new key holding a hash is created.

http://redis.io/commands/hmset
*/
func (self *Client) HMSet(key string, values ...interface{}) (string, error) {
	var ret string

	if len(values)%2 != 0 {
		return "", ErrExpectingPairs
	}

	args := make([][]byte, len(values)+2)
	args[0] = []byte("HMSET")
	args[1] = []byte(key)

	for i, value := range values {
		args[2+i] = to.Bytes(value)
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Sets field in the hash stored at key to value. If key does not exist, a new key
holding a hash is created. If field already exists in the hash, it is
overwritten.

http://redis.io/commands/hset
*/
func (self *Client) HSet(key string, field string, value string) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("HSET"),
		[]byte(key),
		[]byte(field),
		[]byte(value),
	)

	return ret, err
}

/*
Sets field in the hash stored at key to value, only if field does not yet exist.
If key does not exist, a new key holding a hash is created. If field already
exists, this operation has no effect.

http://redis.io/commands/hsetnx
*/
func (self *Client) HSetNX(key string, field string, value interface{}) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("HSETNX"),
		[]byte(key),
		[]byte(field),
		to.Bytes(value),
	)

	return ret, err
}

/*
Returns all values in the hash stored at key.

http://redis.io/commands/hvals
*/
func (self *Client) HVals(key string) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("HVALS"),
		[]byte(key),
	)

	return ret, err
}

/*
Increments the number stored at key by one. If the key does not exist, it is set
to 0 before performing the operation. An error is returned if the key contains a
value of the wrong type or contains a string that can not be represented as
integer. This operation is limited to 64 bit signed integers.

http://redis.io/commands/incr
*/
func (self *Client) Incr(key string) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("INCR"),
		[]byte(key),
	)
	return ret, err
}

/*
Increments the number stored at key by increment. If the key does not exist, it
is set to 0 before performing the operation. An error is returned if the key
contains a value of the wrong type or contains a string that can not be
represented as integer. This operation is limited to 64 bit signed integers.

http://redis.io/commands/incrby
*/
func (self *Client) IncrBy(key string, increment int64) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("INCRBY"),
		[]byte(key),
		to.Bytes(increment),
	)
	return ret, err
}

/*
Increment the string representing a floating point number stored at key by the
specified increment. If the key does not exist, it is set to 0 before
performing the operation.

http://redis.io/commands/incrbyfloat
*/
func (self *Client) IncrByFloat(key string, increment float64) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("INCRBYFLOAT"),
		[]byte(key),
		to.Bytes(increment),
	)
	return ret, err
}

/*
The INFO command returns information and statistics about the server in a format
that is simple to parse by computers and easy to read by humans.

http://redis.io/commands/info
*/
func (self *Client) Info(section string) ([]string, error) {
	var ret []string
	err := self.command(
		&ret,
		[]byte("INFO"),
		[]byte(section),
	)
	return ret, err
}

/*
Returns all keys matching pattern.

http://redis.io/commands/keys
*/
func (self *Client) Keys(pattern string) ([]string, error) {
	var ret []string
	err := self.command(
		&ret,
		[]byte("KEYS"),
		[]byte(pattern),
	)
	return ret, err
}

/*
Return the UNIX TIME of the last DB save executed with success.

http://redis.io/commands/lastsave
*/
func (self *Client) LastSave() (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("LASTSAVE"),
	)
	return ret, err
}

/*
Returns the element at index index in the list stored at key. The index is
zero-based, so 0 means the first element, 1 the second element and so on.
Negative indices can be used to designate elements starting at the tail of the
list. Here, -1 means the last element, -2 means the penultimate and so forth.

http://redis.io/commands/lindex
*/
func (self *Client) LIndex(key string, index int64) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("LINDEX"),
		[]byte(key),
		to.Bytes(index),
	)
	return ret, err
}

/*
Inserts value in the list stored at key either before or after the reference
value pivot.

http://redis.io/commands/linsert
*/
func (self *Client) LInsert(key string, where string, pivot interface{}, value interface{}) (int64, error) {
	var ret int64

	where = strings.ToUpper(where)

	if where != "AFTER" && where != "BEFORE" {
		return 0, errors.New(`The "where" value must be either BEFORE or AFTER.`)
	}

	err := self.command(
		&ret,
		[]byte("LINSERT"),
		[]byte(key),
		[]byte(where),
		to.Bytes(pivot),
		to.Bytes(value),
	)

	return ret, err
}

/*
Returns the length of the list stored at key. If key does not exist, it is
interpreted as an empty list and 0 is returned. An error is returned when the
value stored at key is not a list.

http://redis.io/commands/llen
*/
func (self *Client) LLen(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("LLEN"),
		[]byte(key),
	)

	return ret, err
}

/*
Removes and returns the first element of the list stored at key.

http://redis.io/commands/lpop
*/
func (self *Client) LPop(key string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("LPOP"),
		[]byte(key),
	)

	return ret, err
}

/*
Insert all the specified values at the head of the list stored at key. If key
does not exist, it is created as empty list before performing the push
operations. When key holds a value that is not a list, an error is returned.

http://redis.io/commands/lpush
*/
func (self *Client) LPush(key string, values ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(values)+2)
	args[0] = []byte("LPUSH")
	args[1] = []byte(key)

	for i, _ := range values {
		args[2+i] = to.Bytes(values[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Inserts value at the head of the list stored at key, only if key already exists
and holds a list. In contrary to LPUSH, no operation will be performed when key
does not yet exist.

http://redis.io/commands/lpushx
*/
func (self *Client) LPushX(key string, value interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("LPUSHX"),
		[]byte(key),
		to.Bytes(value),
	)

	return ret, err
}

/*
Returns the specified elements of the list stored at key. The offsets start and
stop are zero-based indexes, with 0 being the first element of the list (the
head of the list), 1 being the next element and so on.

http://redis.io/commands/lrange
*/
func (self *Client) LRange(key string, start int64, stop int64) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("LRANGE"),
		[]byte(key),
		to.Bytes(start),
		to.Bytes(stop),
	)

	return ret, err
}

/*
Removes the first count occurrences of elements equal to value from the list
stored at key. The count argument influences the operation in the following
ways:

http://redis.io/commands/lrem
*/
func (self *Client) LRem(key string, count int64, value interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("LREM"),
		[]byte(key),
		to.Bytes(count),
		to.Bytes(value),
	)

	return ret, err
}

/*
Sets the list element at index to value. For more information on the index
argument, see LINDEX.

http://redis.io/commands/lset
*/
func (self *Client) LSet(key string, index int64, value interface{}) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("LSET"),
		[]byte(key),
		to.Bytes(index),
		to.Bytes(value),
	)

	return ret, err
}

/*
Trim an existing list so that it will contain only the specified range of
elements specified. Both start and stop are zero-based indexes, where 0 is the
first element of the list (the head), 1 the next element and so on.

http://redis.io/commands/ltrim
*/
func (self *Client) LTrim(key string, start int64, stop int64) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("LTRIM"),
		[]byte(key),
		to.Bytes(start),
		to.Bytes(stop),
	)

	return ret, err
}

/*
Returns the values of all specified keys. For every key that does not hold a
string value or does not exist, the special value nil is returned. Because of
this, the operation never fails.

*/
func (self *Client) MGet(keys ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(keys)+1)
	args[0] = []byte("MGET")

	for i, _ := range keys {
		args[1+i] = []byte(keys[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Atomically transfer a key from a source Redis instance to a destination Redis
instance. On success the key is deleted from the original instance and is
guaranteed to exist in the target instance.

http://redis.io/commands/migrate
*/
func (self *Client) Migrate(host string, port uint, key string, destination string, timeout int64) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("MIGRATE"),
		[]byte(host),
		to.Bytes(port),
		[]byte(key),
		[]byte(destination),
		to.Bytes(timeout),
	)

	return ret, err
}

/*
Move key from the currently selected database (see SELECT) to the specified
destination database. When key already exists in the destination database, or
it does not exist in the source database, it does nothing. It is possible to
use MOVE as a locking primitive because of this.

http://redis.io/commands/move
*/
func (self *Client) Move(key string, db string) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("MOVE"),
		[]byte(key),
		[]byte(db),
	)

	return ret, err
}

/*
Sets the given keys to their respective values. MSET replaces existing values
with new values, just as regular SET. See MSETNX if you don't want to overwrite
existing values.

http://redis.io/commands/mset
*/
func (self *Client) MSet(values ...interface{}) (string, error) {
	var ret string

	if len(values)%2 != 0 {
		return "", ErrExpectingPairs
	}

	args := make([][]byte, len(values)+1)
	args[0] = []byte("MSET")

	for i, _ := range values {
		args[1+i] = to.Bytes(values[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Sets the given keys to their respective values. MSETNX will not perform any
operation at all even if just a single key already exists.

http://redis.io/commands/msetnx
*/
func (self *Client) MSetNX(values ...interface{}) (bool, error) {
	var ret bool

	if len(values)%2 != 0 {
		return false, ErrExpectingPairs
	}

	args := make([][]byte, len(values)+1)
	args[0] = []byte("MSETNX")

	for i, _ := range values {
		args[1+i] = to.Bytes(values[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Marks the start of a transaction block. Subsequent commands will be queued for
atomic execution using EXEC.

http://redis.io/commands/multi
*/
func (self *Client) Multi() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("MULTI"),
	)

	return ret, err
}

/*
The OBJECT command allows to inspect the internals of Redis Objects associated
with keys. It is useful for debugging or to understand if your keys are using
the specially encoded data types to save space. Your application may also use
the information reported by the OBJECT command to implement application level
key eviction policies when using Redis as a Cache.

http://redis.io/commands/object
*/
func (self *Client) Object(subcommand string, arguments ...interface{}) (string, error) {
	var ret string

	args := make([][]byte, len(arguments)+2)
	args[0] = []byte("OBJECT")
	args[1] = []byte(subcommand)

	for i, arg := range arguments {
		args[2+i] = to.Bytes(arg)
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Remove the existing timeout on key, turning the key from volatile (a key with an
expire set) to persistent (a key that will never expire as no timeout is
associated).

http://redis.io/commands/persist
*/
func (self *Client) Persist(key string) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("PERSIST"),
		[]byte(key),
	)

	return ret, err
}

/*
This command works exactly like EXPIRE but the time to live of the key is
specified in milliseconds instead of seconds.

http://redis.io/commands/pexpire
*/
func (self *Client) PExpire(key string, milliseconds int64) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("PEXPIRE"),
		[]byte(key),
		to.Bytes(milliseconds),
	)

	return ret, err
}

/*
PEXPIREAT has the same effect and semantic as EXPIREAT, but the Unix time at
which the key will expire is specified in milliseconds instead of seconds.

http://redis.io/commands/pexpireat
*/
func (self *Client) PExpireAt(key string, milliseconds int64) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("PEXPIREAT"),
		[]byte(key),
		to.Bytes(milliseconds),
	)

	return ret, err
}

/*
Returns PONG. This command is often used to test if a connection is still alive,
or to measure latency.

http://redis.io/commands/ping
*/
func (self *Client) Ping() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("PING"),
	)

	return ret, err
}

/*
PSETEX works exactly like SETEX with the sole difference that the expire time is
specified in milliseconds instead of seconds.

http://redis.io/commands/psetex
*/
func (self *Client) PSetEx(key string, milliseconds int64, value interface{}) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("PSETEX"),
		[]byte(key),
		to.Bytes(milliseconds),
		to.Bytes(value),
	)

	return ret, err
}

/*
Subscribes the client to the given patterns.

http://redis.io/commands/psubscribe
*/
func (self *Client) PSubscribe(c chan []string, channel ...string) error {
	var ret []string

	args := make([][]byte, len(channel)+1)
	args[0] = []byte("PSUBSCRIBE")

	for i, _ := range channel {
		args[1+i] = to.Bytes(channel[i])
	}

	err := self.bcommand(c, &ret, args...)

	return err
}

/*
Like TTL this command returns the remaining time to live of a key that has an
expire set, with the sole difference that TTL returns the amount of remaining
time in seconds while PTTL returns it in milliseconds.

http://redis.io/commands/pttl
*/
func (self *Client) PTTL(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("PTTL"),
		[]byte(key),
	)

	return ret, err
}

/*
Posts a message to the given channel.

http://redis.io/commands/publish
*/
func (self *Client) Publish(channel string, message interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("PUBLISH"),
		[]byte(channel),
		to.Bytes(message),
	)

	return ret, err
}

/*
Unsubscribes the client from the given patterns, or from all of them if none is
given.

http://redis.io/commands/punsubscribe
*/
func (self *Client) PUnsubscribe(pattern ...string) (string, error) {
	var ret string

	self.loop = false

	args := make([][]byte, len(pattern)+1)
	args[0] = []byte("PUNSUBSCRIBE")

	for i, _ := range pattern {
		args[1+i] = to.Bytes(pattern[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Ask the server to close the connection. The connection is closed as soon as all
pending replies have been written to the client.

http://redis.io/commands/quit
*/
func (self *Client) Quit() (string, error) {

	self.loop = false

	if self.ctx != nil {

		if self.async != nil {
			// Forcing async connection to clean without waiting for anything else.
			C.redisForceAsyncFree(self.async)
		} else {

			// Request the server to close the connection.
			self.command(
				nil,
				[]byte("QUIT"),
			)

			C.redisFree(self.ctx)
		}

		self.async = nil
		self.ctx = nil

		return "", nil
	}

	return "", ErrNotConnected
}

/*
Return a random key from the currently selected database.

http://redis.io/commands/randomkey
*/
func (self *Client) RandomKey() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("RANDOMKEY"),
	)

	return ret, err
}

/*
Renames key to newkey. It returns an error when the source and destination names
are the same, or when key does not exist. If newkey already exists it is
overwritten.

http://redis.io/commands/rename
*/
func (self *Client) Rename(key string, newkey string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("RENAME"),
		[]byte(key),
		[]byte(newkey),
	)

	return ret, err
}

/*
Renames key to newkey if newkey does not yet exist. It returns an error under
the same conditions as RENAME.

http://redis.io/commands/renamenx
*/
func (self *Client) RenameNX(key string, newkey string) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("RENAMENX"),
		[]byte(key),
		[]byte(newkey),
	)

	return ret, err
}

/*
Create a key associated with a value that is obtained by deserializing the
provided serialized value (obtained via DUMP).

http://redis.io/commands/restore
*/
func (self *Client) Restore(key string, ttl int64, serializedValue string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("RESTORE"),
		[]byte(key),
		to.Bytes(ttl),
		[]byte(serializedValue),
	)

	return ret, err
}

/*
Removes and returns the last element of the list stored at key.

http://redis.io/commands/restore
*/
func (self *Client) RPop(key string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("RPOP"),
		[]byte(key),
	)

	return ret, err
}

/*
Atomically returns and removes the last element (tail) of the list stored at
source, and pushes the element at the first element (head) of the list stored
at destination.

http://redis.io/commands/rpoplpush
*/
func (self *Client) RPopLPush(source string, destination string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("RPOPLPUSH"),
		[]byte(source),
		[]byte(destination),
	)

	return ret, err
}

/*
Insert all the specified values at the tail of the list stored at key. If key
does not exist, it is created as empty list before performing the push
operation. When key holds a value that is not a list, an error is returned.

http://redis.io/commands/rpush
*/
func (self *Client) RPush(key string, values ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(values)+2)
	args[0] = []byte("RPUSH")
	args[1] = []byte(key)

	for i, _ := range values {
		args[2+i] = to.Bytes(values[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Inserts value at the tail of the list stored at key, only if key already exists
and holds a list. In contrary to RPUSH, no operation will be performed when key
does not yet exist.

http://redis.io/commands/rpushx
*/
func (self *Client) RPushX(key string, value interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("RPUSHX"),
		[]byte(key),
		to.Bytes(value),
	)

	return ret, err
}

/*
Add the specified members to the set stored at key. Specified members that are
already a member of this set are ignored. If key does not exist, a new set is
created before adding the specified members.

http://redis.io/commands/sadd
*/
func (self *Client) SAdd(key string, member ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(member)+2)
	args[0] = []byte("SADD")
	args[1] = []byte(key)

	for i, _ := range member {
		args[2+i] = to.Bytes(member[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
The SAVE commands performs a synchronous save of the dataset producing a point
in time snapshot of all the data inside the Redis instance, in the form of an
RDB file.

http://redis.io/commands/save
*/
func (self *Client) Save() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SAVE"),
	)

	return ret, err
}

/*
Returns the set cardinality (number of elements) of the set stored at key.

http://redis.io/commands/scard
*/
func (self *Client) SCard(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("SCARD"),
		[]byte(key),
	)

	return ret, err
}

/*
Returns information about the existence of the scripts in the script cache.

http://redis.io/commands/script-exists
*/
func (self *Client) ScriptExists(script ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(script)+2)
	args[0] = []byte("SCRIPT")
	args[1] = []byte("EXISTS")

	for i, _ := range script {
		args[2+i] = []byte(script[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Flush the Lua scripts cache.

http://redis.io/commands/script-flush
*/
func (self *Client) ScriptFlush() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SCRIPT"),
		[]byte("FLUSH"),
	)

	return ret, err

}

/*
Kills the currently executing Lua script, assuming no write operation was yet
performed by the script.

http://redis.io/commands/script-kill
*/
func (self *Client) ScriptKill() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SCRIPT"),
		[]byte("KILL"),
	)

	return ret, err
}

/*
EVAL and EVALSHA are used to evaluate scripts using the Lua interpreter built
into Redis starting from version 2.6.0.

http://redis.io/commands/eval
*/
func (self *Client) Eval(script string, numkeys int64, arguments ...interface{}) ([]string, error) {
	var ret []string

	args := make([][]byte, len(arguments)+3)
	args[0] = []byte("EVAL")
	args[1] = []byte(script)
	args[2] = to.Bytes(numkeys)

	for i, _ := range arguments {
		args[3+i] = to.Bytes(arguments[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Evaluates a script cached on the server side by its SHA1 digest. Scripts are
cached on the server side using the SCRIPT LOAD command. The command is
otherwise identical to EVAL.

http://redis.io/commands/evalsha
*/
func (self *Client) EvalSHA(hash string, numkeys int64, arguments ...interface{}) ([]string, error) {
	var ret []string

	args := make([][]byte, len(arguments)+3)
	args[0] = []byte("EVALSHA")
	args[1] = []byte(hash)
	args[2] = to.Bytes(numkeys)

	for i, _ := range arguments {
		args[3+i] = to.Bytes(arguments[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Load a script into the scripts cache, without executing it. After the specified
command is loaded into the script cache it will be callable using EVALSHA with
the correct SHA1 digest of the script, exactly like after the first successful
invocation of EVAL.

http://redis.io/commands/script-load
*/
func (self *Client) ScriptLoad(script string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SCRIPT"),
		[]byte("LOAD"),
		[]byte(script),
	)

	return ret, err
}

/*
Returns the members of the set resulting from the difference between the first
set and all the successive sets.

http://redis.io/commands/sdiff
*/
func (self *Client) SDiff(key ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(key)+1)
	args[0] = []byte("SDIFF")

	for i, _ := range key {
		args[1+i] = []byte(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
This command is equal to SDIFF, but instead of returning the resulting set, it
is stored in destination.

http://redis.io/commands/sdiffstore
*/
func (self *Client) SDiffStore(destination string, key ...string) (int64, error) {
	var ret int64

	args := make([][]byte, len(key)+2)
	args[0] = []byte("SDIFFSTORE")
	args[1] = []byte(destination)

	for i, _ := range key {
		args[2+i] = []byte(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Select the DB with having the specified zero-based numeric index. New
connections always use DB 0.

http://redis.io/commands/select
*/
func (self *Client) Select(index int64) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SELECT"),
		to.Bytes(index),
	)

	return ret, err
}

/*
Set key to hold the string value. If key already holds a value, it is
overwritten, regardless of its type.

http://redis.io/commands/set
*/
func (self *Client) Set(key string, value interface{}) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SET"),
		[]byte(key),
		to.Bytes(value),
	)

	return ret, err
}

/*
Sets or clears the bit at offset in the string value stored at key.

http://redis.io/commands/setbit
*/
func (self *Client) SetBit(key string, offset int64, value int64) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("SETBIT"),
		[]byte(key),
		to.Bytes(offset),
		to.Bytes(value),
	)

	return ret, err
}

/*
Set key to hold the string value and set key to timeout after a given number of
seconds. This command is equivalent to executing the following commands:

http://redis.io/commands/setex
*/
func (self *Client) SetEx(key string, seconds int64, value interface{}) (int, error) {
	var ret int

	err := self.command(
		&ret,
		[]byte("SETEX"),
		[]byte(key),
		to.Bytes(seconds),
		to.Bytes(value),
	)

	return ret, err
}

/*
Set key to hold string value if key does not exist. In that case, it is equal to
SET. When key already holds a value, no operation is performed. SETNX is short
for "SET if N ot e X ists".

http://redis.io/commands/setnx
*/
func (self *Client) SetNX(key string, value interface{}) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("SETNX"),
		[]byte(key),
		to.Bytes(value),
	)

	return ret, err
}

/*
Overwrites part of the string stored at key, starting at the specified offset,
for the entire length of value. If the offset is larger than the current length
of the string at key, the string is padded with zero-bytes to make offset fit.
Non-existing keys are considered as empty strings, so this command will make
sure it holds a string large enough to be able to set value at offset.

http://redis.io/commands/setrange
*/
func (self *Client) SetRange(key string, offset int64, value interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("SETRANGE"),
		[]byte(key),
		to.Bytes(offset),
		to.Bytes(value),
	)

	return ret, err
}

/*
Returns the members of the set resulting from the intersection of all the given sets.

http://redis.io/commands/sinter
*/
func (self *Client) SInter(key ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(key)+1)
	args[0] = []byte("SINTER")

	for i, _ := range key {
		args[1+i] = []byte(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
This command is equal to SINTER, but instead of returning the resulting set, it
is stored in destination.

http://redis.io/commands/sinterstore
*/
func (self *Client) SInterStore(destination string, key ...string) (int64, error) {
	var ret int64

	args := make([][]byte, len(key)+2)
	args[0] = []byte("SINTERSTORE")
	args[1] = []byte(destination)

	for i, _ := range key {
		args[2+i] = []byte(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns if member is a member of the set stored at key.

http://redis.io/commands/sismember
*/
func (self *Client) SIsMember(key string, member interface{}) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("SISMEMBER"),
		[]byte(key),
		to.Bytes(member),
	)

	return ret, err
}

/*
The SLAVEOF command can change the replication settings of a slave on the fly.
If a Redis server is already acting as slave, the command SLAVEOF NO ONE will
turn off the replication, turning the Redis server into a MASTER. In the proper
form SLAVEOF hostname port will make the server a slave of another server
listening at the specified hostname and port.

http://redis.io/commands/slaveof
*/
func (self *Client) SlaveOf(key string, port uint) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SLAVEOF"),
		[]byte(key),
		to.Bytes(port),
	)

	return ret, err
}

/*
This command is used in order to read and reset the Redis slow queries log.

http://redis.io/commands/slowlog
*/
func (self *Client) SlowLog(subcommand string, argument interface{}) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("SLOWLOG"),
		[]byte(subcommand),
		to.Bytes(argument),
	)

	return ret, err
}

/*
Returns all the members of the set value stored at key.

http://redis.io/commands/smembers
*/
func (self *Client) SMembers(key string) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("SMEMBERS"),
		[]byte(key),
	)

	return ret, err
}

/*
Move member from the set at source to the set at destination. This operation is
atomic. In every given moment the element will appear to be a member of source
or destination for other clients.

http://redis.io/commands/smove
*/
func (self *Client) SMove(source string, destination string, member interface{}) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("SMOVE"),
		[]byte(source),
		[]byte(destination),
		to.Bytes(member),
	)

	return ret, err
}

/*
Returns or stores the elements contained in the list, set or sorted set at key.
By default, sorting is numeric and elements are compared by their value
interpreted as double precision floating point number.

http://redis.io/commands/sort
*/
func (self *Client) Sort(key string, arguments ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(arguments)+2)
	args[0] = []byte("SORT")
	args[1] = []byte(key)

	for i, _ := range arguments {
		args[2+i] = to.Bytes(arguments[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Removes and returns a random element from the set value stored at key.

http://redis.io/commands/spop
*/
func (self *Client) SPop(key string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SPOP"),
		[]byte(key),
	)

	return ret, err
}

/*
When called with just the key argument, return a random element from the set
value stored at key.

http://redis.io/commands/srandmember
*/
func (self *Client) SRandMember(key string, count int64) ([]string, error) {
	var ret []string

	err := self.command(
		&ret,
		[]byte("SRANDMEMBER"),
		[]byte(key),
		to.Bytes(count),
	)

	return ret, err
}

/*
Remove the specified members from the set stored at key. Specified members that
are not a member of this set are ignored. If key does not exist, it is treated
as an empty set and this command returns 0.

http://redis.io/commands/srem
*/
func (self *Client) SRem(key string, members ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(members)+2)
	args[0] = []byte("SREM")
	args[1] = []byte(key)

	for i, _ := range members {
		args[2+i] = to.Bytes(members[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns the length of the string value stored at key. An error is returned when
key holds a non-string value.

http://redis.io/commands/strlen
*/
func (self *Client) Strlen(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("STRLEN"),
		[]byte(key),
	)

	return ret, err
}

/*
Subscribes the client to the specified channels.

http://redis.io/commands/subscribe
*/
func (self *Client) Subscribe(c chan []string, channel ...string) error {
	var ret []string

	args := make([][]byte, len(channel)+1)
	args[0] = []byte("SUBSCRIBE")

	for i, _ := range channel {
		args[1+i] = to.Bytes(channel[i])
	}

	err := self.bcommand(c, &ret, args...)

	return err
}

/*
Returns the members of the set resulting from the union of all the given sets.

http://redis.io/commands/sunion
*/
func (self *Client) SUnion(key ...string) ([]string, error) {
	var ret []string

	args := make([][]byte, len(key)+1)
	args[0] = []byte("SUNION")

	for i, _ := range key {
		args[1+i] = []byte(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
This command is equal to SUNION, but instead of returning the resulting set, it
is stored in destination.

http://redis.io/commands/sunionstore
*/
func (self *Client) SUnionStore(destination string, key ...string) (int64, error) {
	var ret int64

	args := make([][]byte, len(key)+2)
	args[0] = []byte("SUNIONSTORE")
	args[1] = []byte(destination)

	for i, _ := range key {
		args[2+i] = []byte(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
http://redis.io/commands/sync
*/
func (self *Client) Sync() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("SYNC"),
	)

	return ret, err
}

/*
The TIME command returns the current server time as a two items lists: a Unix
timestamp and the amount of microseconds already elapsed in the current second.
Basically the interface is very similar to the one of the gettimeofday system
call.

http://redis.io/commands/time
*/
func (self *Client) Time() ([]uint64, error) {
	var ret []uint64

	err := self.command(
		&ret,
		[]byte("TIME"),
	)

	return ret, err
}

/*
Returns the remaining time to live of a key that has a timeout. This
introspection capability allows a Redis client to check how many seconds a given
key will continue to be part of the dataset.

http://redis.io/commands/ttl
*/
func (self *Client) TTL(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("TTL"),
		[]byte(key),
	)

	return ret, err
}

/*
Returns the string representation of the type of the value stored at key. The
different types that can be returned are: string, list, set, zset and hash.

http://redis.io/commands/type
*/
func (self *Client) Type(key string) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("TYPE"),
		[]byte(key),
	)

	return ret, err
}

/*
Unsubscribes the client from the given channels, or from all of them if none is
given.

http://redis.io/commands/unsubscribe
*/
func (self *Client) Unsubscribe(channel ...string) error {

	self.loop = false

	args := make([][]byte, len(channel)+1)
	args[0] = []byte("UNSUBSCRIBE")

	for i, _ := range channel {
		args[1+i] = to.Bytes(channel[i])
	}

	err := self.command(nil, args...)

	return err
}

/*
Flushes all the previously watched keys for a transaction.

http://redis.io/commands/unwatch
*/
func (self *Client) Unwatch() (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("UNWATCH"),
	)

	return ret, err
}

/*
Marks the given keys to be watched for conditional execution of a transaction.

http://redis.io/commands/watch
*/
func (self *Client) Watch(key ...string) (string, error) {
	var ret string

	args := make([][]byte, len(key)+1)
	args[0] = []byte("WATCH")

	for i, _ := range key {
		args[1+i] = to.Bytes(key[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Adds all the specified members with the specified scores to the sorted set
stored at key. It is possible to specify multiple score/member pairs. If a
specified member is already a member of the sorted set, the score is updated and
the element reinserted at the right position to ensure the correct ordering. If
key does not exist, a new sorted set with the specified members as sole members
is created, like if the sorted set was empty. If the key exists but does not
hold a sorted set, an error is returned.

http://redis.io/commands/zadd
*/
func (self *Client) ZAdd(key string, arguments ...interface{}) (int64, error) {
	var ret int64

	if len(arguments)%2 != 0 {
		return 0, errors.New("Failed to relate SCORE -> MEMBER using the given arguments.")
	}

	args := make([][]byte, len(arguments)+2)
	args[0] = []byte("ZADD")
	args[1] = []byte(key)

	for i, _ := range arguments {
		args[2+i] = to.Bytes(arguments[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns the sorted set cardinality (number of elements) of the sorted set stored at key.

http://redis.io/commands/zcard
*/
func (self *Client) ZCard(key string) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZCARD"),
		[]byte(key),
	)

	return ret, err
}

/*
Returns the number of elements in the sorted set at key with a score between min
and max.

http://redis.io/commands/zcount
*/
func (self *Client) ZCount(key string, min interface{}, max interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZCOUNT"),
		[]byte(key),
		to.Bytes(min),
		to.Bytes(max),
	)

	return ret, err
}

/*
Increments the score of member in the sorted set stored at key by increment. If
member does not exist in the sorted set, it is added with increment as its score
(as if its previous score was 0.0). If key does not exist, a new sorted set with
the specified member as its sole member is created.

http://redis.io/commands/zincrby
*/
func (self *Client) ZIncrBy(key string, increment int64, member interface{}) (string, error) {
	var ret string

	err := self.command(
		&ret,
		[]byte("ZINCRBY"),
		[]byte(key),
		to.Bytes(increment),
		to.Bytes(member),
	)

	return ret, err
}

/*
Computes the intersection of numkeys sorted sets given by the specified keys,
and stores the result in destination. It is mandatory to provide the number of
input keys (numkeys) before passing the input keys and the other (optional)
arguments.

http://redis.io/commands/zinterstore
*/
func (self *Client) ZInterStore(destination string, numkeys int64, arguments ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(arguments)+3)
	args[0] = []byte("ZINTERSTORE")
	args[1] = []byte(destination)
	args[2] = to.Bytes(numkeys)

	for i, _ := range arguments {
		args[3+i] = to.Bytes(arguments[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns the specified range of elements in the sorted set stored at key. The
elements are considered to be ordered from the lowest to the highest score.
Lexicographical order is used for elements with equal score.

http://redis.io/commands/zrange
*/
func (self *Client) ZRange(key string, values ...interface{}) ([]string, error) {
	var ret []string

	args := make([][]byte, len(values)+2)
	args[0] = []byte("ZRANGE")
	args[1] = []byte(key)

	for i, v := range values {
		args[2+i] = to.Bytes(v)
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns all the elements in the sorted set at key with a score between min and
max (including elements with score equal to min or max). The elements are
considered to be ordered from low to high scores.

http://redis.io/commands/zrangebyscore
*/
func (self *Client) ZRangeByScore(key string, values ...interface{}) ([]string, error) {
	var ret []string

	args := make([][]byte, len(values)+2)
	args[0] = []byte("ZRANGEBYSCORE")
	args[1] = []byte(key)

	for i, _ := range values {
		args[2+i] = to.Bytes(values[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns the rank of member in the sorted set stored at key, with the scores
ordered from low to high. The rank (or index) is 0-based, which means that the
member with the lowest score has rank 0.

http://redis.io/commands/zrank
*/
func (self *Client) ZRank(key string, member interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZRANK"),
		[]byte(key),
		to.Bytes(member),
	)

	return ret, err
}

/*
Removes the specified members from the sorted set stored at key. Non existing
members are ignored.

http://redis.io/commands/zrem
*/
func (self *Client) ZRem(key string, arguments ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(arguments)+2)
	args[0] = []byte("ZREM")
	args[1] = []byte(key)

	for i, _ := range arguments {
		args[2+i] = to.Bytes(arguments[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Removes all elements in the sorted set stored at key with rank between start and
stop. Both start and stop are 0 -based indexes with 0 being the element with the
lowest score. These indexes can be negative numbers, where they indicate offsets
starting at the element with the highest score. For example: -1 is the element
with the highest score, -2 the element with the second highest score and so
forth.

http://redis.io/commands/zremrangebyrank
*/
func (self *Client) ZRemRangeByRank(key string, start interface{}, stop interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZREMRANGEBYRANK"),
		[]byte(key),
		to.Bytes(start),
		to.Bytes(stop),
	)

	return ret, err
}

/*
Removes all elements in the sorted set stored at key with a score between min
and max (inclusive).

http://redis.io/commands/zremrangebyscore
*/
func (self *Client) ZRemRangeByScore(key string, min interface{}, max interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZREMRANGEBYSCORE"),
		[]byte(key),
		to.Bytes(min),
		to.Bytes(max),
	)

	return ret, err
}

/*
Returns the specified range of elements in the sorted set stored at key. The
elements are considered to be ordered from the highest to the lowest score.
Descending lexicographical order is used for elements with equal score.

http://redis.io/commands/zrevrange
*/
func (self *Client) ZRevRange(key string, start int64, stop int64, params ...interface{}) ([]string, error) {
	var ret []string

	args := make([][]byte, len(params)+4)
	args[0] = []byte("ZREVRANGE")
	args[1] = []byte(key)
	args[2] = to.Bytes(start)
	args[3] = to.Bytes(stop)

	for i, v := range params {
		args[4+i] = to.Bytes(v)
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns all the elements in the sorted set at key with a score between max and
min (including elements with score equal to max or min). In contrary to the
default ordering of sorted sets, for this command the elements are considered to
be ordered from high to low scores.

http://redis.io/commands/zrevrangebyscore
*/
func (self *Client) ZRevRangeByScore(key string, start interface{}, stop interface{}, params ...interface{}) ([]string, error) {
	var ret []string

	args := make([][]byte, len(params)+4)
	args[0] = []byte("ZREVRANGEBYSCORE")
	args[1] = []byte(key)
	args[2] = to.Bytes(start)
	args[3] = to.Bytes(stop)

	for i, _ := range params {
		args[4+i] = to.Bytes(params[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}

/*
Returns the rank of member in the sorted set stored at key, with the scores
ordered from high to low. The rank (or index) is 0-based, which means that the
member with the highest score has rank 0.

http://redis.io/commands/zrevrank
*/
func (self *Client) ZRevRank(key string, member interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZREVRANK"),
		[]byte(key),
		to.Bytes(member),
	)

	return ret, err
}

/*
Returns the score of member in the sorted set at key.

http://redis.io/commands/zscore
*/
func (self *Client) ZScore(key string, member interface{}) (int64, error) {
	var ret int64

	err := self.command(
		&ret,
		[]byte("ZSCORE"),
		[]byte(key),
		to.Bytes(member),
	)

	return ret, err
}

/*
Computes the union of numkeys sorted sets given by the specified keys, and
stores the result in destination. It is mandatory to provide the number of input
keys (numkeys) before passing the input keys and the other (optional) arguments.

http://redis.io/commands/zunionstore
*/
func (self *Client) ZUnionStore(destination string, numkeys int64, key string, params ...interface{}) (int64, error) {
	var ret int64

	args := make([][]byte, len(params)+4)
	args[0] = []byte("ZUNIONSTORE")
	args[1] = []byte(destination)
	args[2] = to.Bytes(numkeys)
	args[3] = []byte(key)

	for i, _ := range params {
		args[4+i] = to.Bytes(params[i])
	}

	err := self.command(&ret, args...)

	return ret, err
}
