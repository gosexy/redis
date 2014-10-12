// Copyright (c) 2013-2014 José Carlos Nieto, https://menteslibres.net/xiam
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// The `redis` package is yet another Go client library for redis-server.
package redis

import (
	"errors"
	"fmt"
	"menteslibres.net/gosexy/to"
	"time"
)

var defaultTimeout = time.Second * 20

const (
	// Default port for connecting to redis-server.
	DefaultPort = 6379
)

// Error messages
var (
	ErrNilReply        = errors.New(`Received a nil response.`)
	ErrNotConnected    = errors.New(`Client is not connected.`)
	ErrExpectingPairs  = errors.New(`Expecting (field -> value) pairs.`)
	ErrNotInitialized  = errors.New(`Client is not initialized. Forgot to use redis.New()?`)
	ErrUnknownResponse = errors.New(`Unknown response.`)
	ErrNotEnoughData   = errors.New(`Not enough data.`)
)

// A redis client
type Client struct {
	redis *conn
}

func New() *Client {
	c := new(Client)
	return c
}

func (c *Client) Close() error {
	if c == nil {
		return ErrNotConnected
	}
	if c.redis != nil {
		c.redis.close()
		c.redis = nil
	}
	return nil
}

// Connects the client to the given host and port.
func (c *Client) Connect(host string, port uint) (err error) {
	return c.dial(`tcp`, fmt.Sprintf(`%s:%d`, host, port))
}

// ConnectWithTimeout attempts to connect to a redis-server, giving up after
// the specified time.
func (c *Client) ConnectWithTimeout(host string, port uint, timeout time.Duration) error {
	return c.dialTimeout(`tcp`, fmt.Sprintf(`%s:%d`, host, port), timeout)
}

// ConnectNonBlock connects the client to the given host and port (deprecated).
func (c *Client) ConnectNonBlock(host string, port uint) error {
	return c.Connect(host, port)
}

// ConnectUnixNonBlock attempts to create a non-blocking connection with an
// UNIX socket (deprecated).
func (c *Client) ConnectUnixNonBlock(path string) error {
	return c.ConnectUnix(path)
}

// ConnectUnix attempts to create a connection with a UNIX socket.
func (c *Client) ConnectUnix(path string) error {
	return c.dial(`unix`, path)
}

// ConnectUnixWithTimeout attempts to create a connection with an UNIX socket,
// giving up after the specified time.
func (c *Client) ConnectUnixWithTimeout(path string, timeout time.Duration) error {
	return c.dialTimeout(`unix`, path, timeout)
}

func (c *Client) dialTimeout(network, address string, timeout time.Duration) error {
	var err error

	if c == nil {
		return ErrNotInitialized
	}

	if c.redis != nil {
		c.Close()
	}

	if timeout == 0 {
		timeout = defaultTimeout
	}

	if c.redis, err = dialTimeout(network, address, timeout); err != nil {
		return err
	}

	return nil
}

func (c *Client) dial(network, address string) error {
	var err error

	if c == nil {
		return ErrNotInitialized
	}

	if c.redis != nil {
		c.Close()
	}

	if c.redis, err = dial(network, address); err != nil {
		return err
	}

	return nil
}

// Command builds a command specified by the `values` interface and stores the
// result into the variable pointed by `dest`.
func (c *Client) Command(dest interface{}, values ...interface{}) error {
	if c == nil {
		return ErrNotInitialized
	}
	bvalues := make([][]byte, len(values))
	// Converting all input values into []byte.
	for i := range values {
		bvalues[i] = to.Bytes(values[i])
	}
	// Sending command to redis-server.
	return c.command(dest, bvalues...)
}

func (c *Client) bcommand(cn chan []string, values ...[]byte) error {
	return c.redis.blockCommand(cn, values...)
}

func (c *Client) command(dest interface{}, values ...[]byte) error {
	return c.redis.syncCommand(dest, values...)
}
