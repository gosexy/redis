/*
  Copyright (c) 2013 José Carlos Nieto, http://xiam.menteslibres.org/

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

package redis

/*
#cgo CFLAGS: -I./_hiredis

#include <stdarg.h>

#include "net.h"
#include "net.c"

#include "sds.h"
#include "sds.c"

#include "hiredis.h"
#include "hiredis.c"

struct timeval redisTimeVal(long sec, long usec) {
	struct timeval t = { sec, usec };
	return t;
}

int redisGetReplyType(redisReply *r) {
	return r->type;
}

redisReply *redisReplyGetElement(redisReply *el, int i) {
	return el->element[i];
}

*/
import "C"

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
	//"unsafe"
)

func timeVal(timeout time.Duration) C.struct_timeval {
	return C.redisTimeVal(C.long(int64(timeout/time.Second)), C.long(int64(timeout%time.Millisecond)))
}

type Client struct {
	ctx *C.redisContext
}

func New() *Client {
	self := &Client{}
	return self
}

func byteValue(v interface{}) []byte {
	return []byte(fmt.Sprintf("%v", v))
}

func (self *Client) ConnectWithTimeout(host string, port uint, timeout time.Duration) error {
	var ctx *C.redisContext

	ctx = C.redisConnectWithTimeout(
		C.CString(host),
		C.int(port),
		timeVal(timeout),
	)
	if ctx == nil {
		return fmt.Errorf("Could not allocate redis context.")
	} else if ctx.err > 0 {
		return fmt.Errorf("Could not connect: %s", ctx.errstr)
	}

	self.ctx = ctx

	return nil
}

func setReplyValue(v reflect.Value, reply *C.redisReply) error {

	switch C.redisGetReplyType(reply) {
	// Received a string.
	case C.REDIS_REPLY_STRING, C.REDIS_REPLY_STATUS, C.REDIS_REPLY_ERROR:
		// Destination type.
		switch v.Kind() {
		case reflect.String:
			// string -> string
			v.Set(reflect.ValueOf(C.GoString(reply.str)))
		case reflect.Int:
			// string -> int
			i, _ := strconv.ParseInt(C.GoString(reply.str), 10, 0)
			v.Set(reflect.ValueOf(int(i)))
		case reflect.Int8:
			// string -> int8
			i, _ := strconv.ParseInt(C.GoString(reply.str), 10, 8)
			v.Set(reflect.ValueOf(int8(i)))
		case reflect.Int16:
			// string -> int16
			i, _ := strconv.ParseInt(C.GoString(reply.str), 10, 16)
			v.Set(reflect.ValueOf(int16(i)))
		case reflect.Int32:
			// string -> int32
			i, _ := strconv.ParseInt(C.GoString(reply.str), 10, 32)
			v.Set(reflect.ValueOf(int32(i)))
		case reflect.Int64:
			// string -> int64
			i, _ := strconv.ParseInt(C.GoString(reply.str), 10, 64)
			v.Set(reflect.ValueOf(int64(i)))
		case reflect.Uint:
			// string -> uint
			i, _ := strconv.ParseUint(C.GoString(reply.str), 10, 0)
			v.Set(reflect.ValueOf(uint(i)))
		case reflect.Uint8:
			// string -> uint8
			i, _ := strconv.ParseUint(C.GoString(reply.str), 10, 8)
			v.Set(reflect.ValueOf(uint8(i)))
		case reflect.Uint16:
			// string -> uint16
			i, _ := strconv.ParseUint(C.GoString(reply.str), 10, 16)
			v.Set(reflect.ValueOf(uint16(i)))
		case reflect.Uint32:
			// string -> uint32
			i, _ := strconv.ParseUint(C.GoString(reply.str), 10, 32)
			v.Set(reflect.ValueOf(uint32(i)))
		case reflect.Uint64:
			// string -> uint64
			i, _ := strconv.ParseUint(C.GoString(reply.str), 10, 64)
			v.Set(reflect.ValueOf(uint64(i)))
		default:
			return fmt.Errorf("Unsupported conversion: redis string to %v", v.Kind())
		}
	// Received integer.
	case C.REDIS_REPLY_INTEGER:
		switch v.Kind() {
		case reflect.String:
			// integer -> string
			v.Set(reflect.ValueOf(fmt.Sprintf("%d", int64(reply.integer))))
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
		}
	case C.REDIS_REPLY_ARRAY:
		switch v.Kind() {
		case reflect.Slice:
			var err error
			total := int(reply.elements)
			elements := reflect.MakeSlice(v.Type(), total, total)
			for i := 0; i < total; i++ {
				item := C.redisReplyGetElement(reply, C.int(i))
				err = setReplyValue(elements.Index(i), item)
				if err != nil {
					return err
				}
			}
			v.Set(elements)
		default:
			return fmt.Errorf("Unsupported conversion: redis string to %v", v.Kind())
		}
	}
	return nil
}

func (self *Client) Command(dest interface{}, values ...interface{}) error {

	argc := len(values)
	argv := make([][]byte, argc)

	for i := 0; i < argc; i++ {
		value := values[i]
		switch value.(type) {
		case string:
			argv[i] = []byte(value.(string))
		case int64:
			argv[i] = []byte(fmt.Sprintf("%d", value.(int64)))
		case int32:
			argv[i] = []byte(fmt.Sprintf("%d", value.(int32)))
		case int16:
			argv[i] = []byte(fmt.Sprintf("%d", value.(int16)))
		case int8:
			argv[i] = []byte(fmt.Sprintf("%d", value.(int8)))
		case int:
			argv[i] = []byte(fmt.Sprintf("%d", value.(int)))
		case uint64:
			argv[i] = []byte(fmt.Sprintf("%d", value.(uint64)))
		case uint32:
			argv[i] = []byte(fmt.Sprintf("%d", value.(uint32)))
		case uint16:
			argv[i] = []byte(fmt.Sprintf("%d", value.(uint16)))
		case uint8:
			argv[i] = []byte(fmt.Sprintf("%d", value.(uint8)))
		case uint:
			argv[i] = []byte(fmt.Sprintf("%d", value.(uint)))
		default:
			return fmt.Errorf("Unsupported input type: %v.", reflect.TypeOf(value))
		}
	}

	return self.command(dest, argv...)
}

func (self *Client) command(dest interface{}, values ...[]byte) error {

	argc := len(values)
	argv := make([](*C.char), argc)
	argvlen := make([]C.size_t, argc)

	for i, _ := range values {
		argv[i] = C.CString(string(values[i]))
		argvlen[i] = C.size_t(len(values[i]))
	}

	raw := C.redisCommandArgv(self.ctx, C.int(argc), &argv[0], (*C.size_t)(&argvlen[0]))
	defer C.freeReplyObject(raw)

	reply := (*C.redisReply)(raw)

	switch C.redisGetReplyType(reply) {
	/*
		case C.REDIS_REPLY_STRING:
		case C.REDIS_REPLY_ARRAY:
		case C.REDIS_REPLY_INTEGER:
		case C.REDIS_REPLY_NIL:
		case C.REDIS_REPLY_STATUS:
	*/
	case C.REDIS_REPLY_ERROR:
		return fmt.Errorf(C.GoString(reply.str))
	}

	if dest == nil {
		return nil
	}

	rv := reflect.ValueOf(dest)

	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("Destination is not a pointer: %v\n", rv)
	}

	return setReplyValue(rv.Elem(), reply)
}

func (self *Client) LRange(name string, min int, max int) ([]string, error) {
	var ret []string
	err := self.command(&ret,
		[]byte("LRANGE"),
		[]byte(string(name)),
		[]byte(strconv.Itoa(min)),
		[]byte(strconv.Itoa(max)),
	)
	return ret, err
}

func (self *Client) Set(name string, value string) (string, error) {
	var s string
	err := self.command(&s,
		[]byte(string("SET")),
		[]byte(string(name)),
		[]byte(string(value)),
	)
	return s, err
}

func (self *Client) LPush(name string, value interface{}) (int64, error) {
	var ret int64
	err := self.command(&ret,
		[]byte(string("LPUSH")),
		[]byte(string(name)),
		byteValue(value),
	)
	return ret, err
}

func (self *Client) Ping() (string, error) {
	var s string
	err := self.command(&s, []byte("PING"))
	return s, err
}

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
		byteValue(value),
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
func (self *Client) BitCount(key string, start int64, end int64) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("BITCOUNT"),
		byteValue(start),
		byteValue(end),
	)
	return ret, err
}

/*
Perform a bitwise operation between multiple keys (containing string values)
and store the result in the destination key.

http://redis.io/commands/bitop
*/
func (self *Client) BitOp(op string, keys ...string) (int64, error) {
	var ret int64
	args := make([][]byte, len(keys)+1)
	args[0] = []byte("BITOP")
	for i, key := range keys {
		args[1+i] = byteValue(key)
	}
	err := self.command(ret, args...)
	return ret, err
}

/*
BLPOP is a blocking list pop primitive. It is the blocking version of LPOP
because it blocks the connection when there are no elements to pop from any of
the given lists. An element is popped from the head of the first list that is
non-empty, with the given keys being checked in the order that they are given.

http://redis.io/commands/blpop
*/
func (self *Client) BlPop(timeout uint64, keys ...string) ([]string, error) {
	var i int
	var ret []string
	args := make([][]byte, len(keys)+2)
	args[0] = []byte("BLPOP")

	for i, key := range keys {
		args[1+i] = byteValue(key)
	}
	args[1+i] = byteValue(timeout)

	err := self.command(ret, args...)
	return ret, err
}

/*
BRPOP is a blocking list pop primitive. It is the blocking version of RPOP
because it blocks the connection when there are no elements to pop from any of
the given lists. An element is popped from the tail of the first list that is
non-empty, with the given keys being checked in the order that they are given.

http://redis.io/commands/brpop
*/
func (self *Client) BrPop(timeout uint64, keys ...string) ([]string, error) {
	var i int
	var ret []string

	args := make([][]byte, len(keys)+2)
	args[0] = []byte("BRPOP")

	for i, key := range keys {
		args[1+i] = byteValue(key)
	}
	args[1+i] = byteValue(timeout)

	err := self.command(ret, args...)
	return ret, err
}

/*
BRPOPLPUSH is the blocking variant of RPOPLPUSH. When source contains elements,
this command behaves exactly like RPOPLPUSH. When source is empty, Redis will
block the connection until another client pushes to it or until timeout is
reached. A timeout of zero can be used to block indefinitely.

http://redis.io/commands/brpoplpush
*/
func (self *Client) BrPopLPush(source string, destination string, timeout int64) ([]string, error) {
	var ret []string
	err := self.command(
		&ret,
		[]byte("BRPOPLPUSH"),
		byteValue(source),
		byteValue(destination),
		byteValue(timeout),
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
		[]byte("CLIENT KILL"),
		byteValue(fmt.Sprintf("%s:%d", ip, port)),
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
		[]byte("CLIENT LIST"),
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
		[]byte("CLIENT GETNAME"),
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
		[]byte("CLIENT SETNAME"),
		byteValue(connectionName),
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
		[]byte("CONFIG GET"),
		byteValue(parameter),
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
		[]byte("CONFIG SET"),
		byteValue(parameter),
		byteValue(value),
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
		[]byte("CONFIG RESETSTAT"),
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
		byteValue(key),
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
		byteValue(key),
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
		byteValue(key),
		byteValue(decrement),
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
		args[1+i] = byteValue(key)
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
		byteValue(key),
	)
	return ret, err
}

/*
Returns message.

http://redis.io/commands/echo
*/
func (self *Client) Echo(message string) (string, error) {
	var ret string
	err := self.command(
		&ret,
		[]byte("ECHO"),
		byteValue(message),
	)
	return ret, err
}

/*
Executes all previously queued commands in a transaction and restores the connection state to normal.

http://redis.io/commands/exec
*/
func (self *Client) Exec() ([]string, error) {
	var ret []string
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
		byteValue(key),
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
		byteValue(key),
		byteValue(seconds),
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
		byteValue(key),
		byteValue(unixTime),
	)
	return ret, err
}

/*
Delete all the keys of all the existing databases, not just the currently selected one. This command never fails.

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
func (self *Client) FlushDb() (string, error) {
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
		[]byte(string("GET")),
		byteValue(key),
	)
	return s, err
}

/*
Returns the bit value at offset in the string value stored at key.

http://redis.io/commands/getbit
*/
func (self *Client) GetBit(key string, offset int64) (int64, error) {
	var s int64
	err := self.command(&s,
		[]byte(string("GETBIT")),
		byteValue(key),
		byteValue(offset),
	)
	return s, err
}

/*
Returns the substring of the string value stored at key, determined by the
offsets start and end (both are inclusive). Negative offsets can be used in
order to provide an offset starting from the end of the string. So -1 means
the last character, -2 the penultimate and so forth.

http://redis.io/commands/getrange
*/
func (self *Client) GetRange(key string, start int64, end int64) ([]string, error) {
	var ret []string
	err := self.command(&ret,
		[]byte("GETRANGE"),
		byteValue(key),
		byteValue(start),
		byteValue(end),
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
		byteValue(key),
		byteValue(value),
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

	args := make([][]byte, len(fields)+1)
	args[0] = []byte("HDEL")

	for i, field := range fields {
		args[1+i] = byteValue(field)
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
		byteValue(increment),
	)

	return ret, err
}

/*
Increment the specified field of an hash stored at key, and representing a
floating point number, by the specified increment. If the field does not exist,
it is set to 0 before performing the operation.

http://redis.io/commands/hincrbyfloat
*/
func (self *Client) HIncrByFloat(key string, field string, increment float64) (float64, error) {
	var ret float64

	err := self.command(
		&ret,
		[]byte("HINCRBYFLOAT"),
		[]byte(key),
		[]byte(field),
		byteValue(increment),
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

	args := make([][]byte, len(fields)+1)
	args[0] = []byte("HMGET")

	for i, field := range fields {
		args[1+i] = byteValue(field)
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
func (self *Client) HMSet(key string, values ...string) (string, error) {
	var ret string

	if len(values)%2 != 0 {
		return "", fmt.Errorf("Expecting a field:value pair.")
	}

	args := make([][]byte, len(values)+1)
	args[0] = []byte("HMSET")

	for i, value := range values {
		args[1+i] = byteValue(value)
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
func (self *Client) HSetNX(key string, field string, value string) (bool, error) {
	var ret bool

	err := self.command(
		&ret,
		[]byte("HSETNX"),
		[]byte(key),
		[]byte(field),
		[]byte(value),
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
func (self *Client) Incr(name string) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("INCR"),
		[]byte(name),
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
func (self *Client) IncrBy(name string, increment int64) (int64, error) {
	var ret int64
	err := self.command(
		&ret,
		[]byte("INCRBY"),
		[]byte(name),
		byteValue(increment),
	)
	return ret, err
}

/*
Increment the string representing a floating point number stored at key by the
specified increment. If the key does not exist, it is set to 0 before
performing the operation.

http://redis.io/commands/incrbyfloat
*/
func (self *Client) IncrByFloat(name string, increment float64) (float64, error) {
	var ret float64
	err := self.command(
		&ret,
		[]byte("INCRBYFLOAT"),
		[]byte(name),
		byteValue(increment),
	)
	return ret, err
}
