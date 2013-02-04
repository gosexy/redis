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

*/
import "C"

import (
	"fmt"
	"time"
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

func (self *Client) Ping() (string, error) {
	var s string

	// void *redisCommandArgv(redisContext *c, int argc, const char **argv, const size_t *argvlen);
	argv := [](*C.char){C.CString("PING")}
	argvlen := []C.size_t{C.size_t(4)}

	reply := C.redisCommandArgv(self.ctx, C.int(1), &argv[0], (*C.size_t)(&argvlen[0]))

	s = C.GoString((*C.redisReply)(reply).str)

	C.freeReplyObject(reply)
	return s, nil
}
