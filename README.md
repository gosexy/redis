# gosexy/redis

This package provides Go bindings for the official [Redis][1] client,
[hiredis][2], and [benchmarking][8] tools for comparing different Go redis
clients.

The complete set of commands for the 2.6.10 release is supported.

## How to install

```
go get menteslibres.net/gosexy/redis
```

## Usage

Install the package with `go get` and use `import` to include it in your project.

```
import "menteslibres.net/gosexy/redis"
```

The `redis.New()` function returns a `*redis.Client` struct that you can then
use to interact with your redis server.

```go
var client *redis.Client

client = redis.New()

err = client.Connect(host, port)

if err != nil {
  log.Fatalf("Connect failed: %s\n", err.Error())
  return
}

log.Println("Connected to redis-server.")

log.Printf("Sending PING...\n")
s, err = client.Ping()

if err != nil {
  log.Fatalf("Could not ping: %s\n", err.Error())
  return
}

log.Printf("Received %s!\n", s)

client.Quit()
```

This is the expected output of the above
[ping-pong example](_examples/test-ping/main.go):

```
2013/02/19 07:15:52 Connected to redis-server.
2013/02/19 07:15:52 Sending PING...
2013/02/19 07:15:52 Received PONG!
```

Remember to use `client.Quit()` to close the client connection.

## Examples

Some examples are included.

* [PING and PONG](_examples/test-ping/main.go)
* [GET, SET, and lists](_examples/test-get-set/main.go)
* [RANGE](_examples/test-range/main.go)
* [Using redis.Client.Command()](_examples/test-custom-commands/main.go)
* [Using SUBSCRIBE](_examples/test-subscribe/main.go)
* [Using SUBSCRIBE, another approach](_examples/test-subscribe-2/main.go)

### Simple SET and GET

An example on how to use `*redis.Client` to send `SET` and `GET`
commands to the client.

```go
var client *redis.Client
var err error

client = redis.New()

err = client.Connect(host, port)

if err != nil {
  log.Fatalf("Connect failed: %s\n", err.Error())
  return
}

log.Println("Connected to redis-server.")

// DEL hello
log.Printf("DEL hello\n")
client.Del("hello")

// SET hello 1
log.Printf("SET hello 1\n")
client.Set("hello", 1)

// INCR hello
log.Printf("INCR hello\n")
client.Incr("hello")

// GET hello
log.Printf("GET hello\n")
s, err = client.Get("hello")

if err != nil {
  log.Fatalf("Could not GET: %s\n", err.Error())
  return
}

log.Printf("> hello = %s\n", s)

client.Quit()
```

This is the expected output of the above
[set-get example](_examples/test-get-set/main.go):

```
2013/02/19 07:19:37 Connected to redis-server.
2013/02/19 07:19:37 DEL hello
2013/02/19 07:19:37 SET hello 1
2013/02/19 07:19:37 INCR hello
2013/02/19 07:19:37 GET hello
2013/02/19 07:19:37 > hello = 2
```

### Subscriptions

You can use `SUBSCRIBE` and `PSUBSCRIBE` with [channels][3] and
[goroutines][4] inside a non-blocking connection. You can create a non-blocking
connection using the `ConnectNonBlock` or `ConnectUnixNonBlock` functions.

```go
go consumer.Subscribe(rec, "channel")

var ls []string

for {
  ls = <-rec
  log.Printf("Consumer received: %v\n", strings.Join(ls, ", "))
}
```

The above snippet is part of a
[subscription example](_examples/test-subscribe-2/main.go), if you run the
full example you'll see something like this:

```
2013/02/19 07:25:33 Consumer received: message, channel, Hello world!
2013/02/19 07:25:33 Consumer received: message, channel, Do you know how to count?
2013/02/19 07:25:33 Consumer received: message, channel, 0
2013/02/19 07:25:33 Consumer received: message, channel, 1
2013/02/19 07:25:33 Consumer received: message, channel, 2
```

## Documentation

See the [online docs][6] for gosexy/redis at [godoc.org][5].

Don't forget to check the [complete list of redis commands][7] too!

## Licenses

This library wraps the awesome `hiredis.c` client that is released under this
license:

```
/*
 * Copyright (c) 2009-2011, Salvatore Sanfilippo <antirez at gmail dot com>
 * Copyright (c) 2010-2011, Pieter Noordhuis <pcnoordhuis at gmail dot com>
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *   * Redistributions of source code must retain the above copyright notice,
 *     this list of conditions and the following disclaimer.
 *   * Redistributions in binary form must reproduce the above copyright
 *     notice, this list of conditions and the following disclaimer in the
 *     documentation and/or other materials provided with the distribution.
 *   * Neither the name of Redis nor the names of its contributors may be used
 *     to endorse or promote products derived from this software without
 *     specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */
```

It also includes portions of code with the following license:

```
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
```

The code that wraps them all, `gosexy/redis`, is released under an equally
friendly license:

> Copyright (c) 2013 José Carlos Nieto, http://xiam.menteslibres.org/
>
> Permission is hereby granted, free of charge, to any person obtaining
> a copy of this software and associated documentation files (the
> "Software"), to deal in the Software without restriction, including
> without limitation the rights to use, copy, modify, merge, publish,
> distribute, sublicense, and/or sell copies of the Software, and to
> permit persons to whom the Software is furnished to do so, subject to
> the following conditions:
>
> The above copyright notice and this permission notice shall be
> included in all copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
> MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
> NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
> LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
> OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
> WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

[1]: http://redis.io
[2]: https://github.com/redis/hiredis
[3]: http://golang.org/doc/effective_go.html#channels
[4]: http://golang.org/doc/effective_go.html#goroutines
[5]: http://godoc.org
[6]: http://godoc.org/menteslibres.net/gosexy/redis
[7]: http://redis.io/commands
[8]: https://github.com/gosexy/redis/tree/master/_benchmarks
