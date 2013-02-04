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

func (self *Client) Del(name string) (int64, error) {
	var s int64
	err := self.command(&s,
		[]byte(string("DEL")),
		[]byte(string(name)),
	)
	return s, err
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

func (self *Client) Get(name string) (string, error) {
	var s string
	err := self.command(&s,
		[]byte(string("GET")),
		[]byte(string(name)),
	)
	return s, err
}

func (self *Client) Incr(name string) (int64, error) {
	var ret int64
	err := self.command(&ret,
		[]byte(string("INCR")),
		[]byte(string(name)),
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

func (self *Client) LPush(name string, value string) (int64, error) {
	var ret int64
	err := self.command(&ret,
		[]byte(string("LPUSH")),
		[]byte(string(name)),
		[]byte(string(value)),
	)
	return ret, err
}

func (self *Client) Ping() (string, error) {
	var s string
	err := self.command(&s, []byte("PING"))
	return s, err
}
