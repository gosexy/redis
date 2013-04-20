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

#include <stdarg.h>
#include <stdlib.h>

#include "net.h"
#include "net.c"

#include "sds.h"
#include "sds.c"

#include "hiredis.h"
#include "hiredis.c"

#include "async.h"
#include "async.c"

#include "_cgo_export.h"

struct timeval redisTimeVal(long sec, long usec) {
	struct timeval t = { sec, usec };
	return t;
}

// Will call Go's asyncReceive.
void redisAsyncCatch(redisAsyncContext *c, void *r, void *privdata) {
	redisReply *reply = r;
	asyncReceive(reply, privdata);
}

// Wraps redisAsyncCommandArgv that calls redisAsyncCatch().
int redisAsyncCommandArgvWrapper(redisAsyncContext *ac, void *fn, void *privdata, int argc, const char **argv, const size_t *argvlen) {
	return redisAsyncCommandArgv(ac, redisAsyncCatch, privdata, argc, argv, argvlen);
}

// Returns the redis reply type.
int redisGetReplyType(redisReply *r) {
	if (r != NULL) {
		return r->type;
	}
	return -1;
}

// Returns the (i)th element of the redisReply.
redisReply *redisReplyGetElement(redisReply *el, int i) {
	return el->element[i];
}

void redisGoroutineReadEvent(void *arg) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)arg;
	redisAsyncContext *c = e->context;
	redisAsyncHandleRead(e->context);
}

void redisGoroutineWriteEvent(void *arg) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)arg;
	redisAsyncHandleWrite(e->context);
}

static void redisGoroutineAddRead(void *privdata) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)privdata;
	redisEventAddRead(e);
}

static void redisGoroutineDelRead(void *privdata) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)privdata;
	redisEventDelRead(e);
}

static void redisGoroutineAddWrite(void *privdata) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)privdata;
	redisEventAddWrite(e);
}

static void redisGoroutineDelWrite(void *privdata) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)privdata;
	redisEventDelWrite(e);
}

static void redisGoroutineCleanup(void *privdata) {
	redisGoroutineEvents *e = (redisGoroutineEvents*)privdata;
	redisEventCleanup(e);
}

int redisGoroutineAttach(redisAsyncContext *ac, struct redisEvent *ev) {
	redisContext *c = &(ac->c);
	redisGoroutineEvents *e;

	/* Nothing should be attached when something is already attached */
	if (ac->ev.data != NULL)
		return REDIS_ERR;

	/* Create container for context and r/w events */
	e = (redisGoroutineEvents*)malloc(sizeof(*e));
	e->context = ac;

	/* Register functions to start/stop listening for events */
	ac->ev.addRead = redisGoroutineAddRead;
	ac->ev.delRead = redisGoroutineDelRead;
	ac->ev.addWrite = redisGoroutineAddWrite;
	ac->ev.delWrite = redisGoroutineDelWrite;
	ac->ev.cleanup = redisGoroutineCleanup;
	ac->ev.data = e;

	return REDIS_OK;
}
