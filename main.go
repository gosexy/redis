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
	"unsafe"
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
	case C.REDIS_REPLY_STRING:
	case C.REDIS_REPLY_ARRAY:
	case C.REDIS_REPLY_INTEGER:
	case C.REDIS_REPLY_NIL:
	case C.REDIS_REPLY_STATUS:
	case C.REDIS_REPLY_ERROR:
		return fmt.Errorf(C.GoString(reply.str))
	}

	rv := reflect.ValueOf(dest)
	pv := rv

	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		return fmt.Errorf("Invalid destination type: %v\n", pv)
	}

	el := rv.Elem()

	switch el.Kind() {
	case reflect.String:
		el.Set(reflect.ValueOf(C.GoString(reply.str)))
	case reflect.Int64:
		el.Set(reflect.ValueOf(int64(reply.integer)))
	case reflect.Slice:
		total := int(reply.elements)
		elements := make([]string, total)
		for i := 0; i < total; i++ {
			item := C.redisReplyGetElement(reply, C.int(i))
			elements[i] = C.GoString(item.str)
			C.freeReplyObject(unsafe.Pointer(item))
		}
		el.Set(reflect.ValueOf(elements))
	default:
		return fmt.Errorf("Unknown receiver kind: %v.", el.Kind())
	}

	return nil
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
