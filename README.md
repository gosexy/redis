# menteslibres.net/gosexy/redis

Package `redis` was formerly a wrapper for the official redis C client
[hiredis][10]. As of October 2014, package `redis` was splitted into two
different packages: a [RESP decoder][8] (`resp`) and the [redis client][9]
(`redis`) this page describes.

[![Build Status](https://travis-ci.org/gosexy/redis.png)](https://travis-ci.org/gosexy/redis)

## How to install or upgrade

Use `go get` to install or upgrade (`-u`) the `redis` package.

```
go get -u menteslibres.net/gosexy/redis
```

## Usage

Use `import` to use `redis` in your program:

```
import (
  "menteslibres.net/gosexy/redis"
)
```

The `redis.New()` function returns a `*redis.Client` pointer that you can use
to interact with a redis server.

This example shows how to connect and ping a redis server.

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

Always use `client.Quit()` to close the client connection.

## Examples

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

## License

> Copyright (c) 2013-2014 José Carlos Nieto, https://menteslibres.net/xiam
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
[3]: http://golang.org/doc/effective_go.html#channels
[4]: http://golang.org/doc/effective_go.html#goroutines
[5]: http://godoc.org
[6]: http://godoc.org/menteslibres.net/gosexy/redis
[7]: http://redis.io/commands
[8]: https://github.com/xiam/resp
[9]: https://github.com/gosexy/redis
[10]: https://github.com/redis/hiredis
