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

const (
	// Default port for connecting to redis-server.
	DefaultPort = 6379
)

// Error messages
var (
	ErrIO                  = errors.New(`Input/output error.`)
	ErrEOF                 = errors.New(`Unexpected EOF.`)
	ErrProtocol            = errors.New(`Protocol error.`)
	ErrOOM                 = errors.New(`Out of memory.`)
	ErrOther               = errors.New(`Unknown error.`)
	ErrFailedAllocation    = errors.New(`Got memory allocation problem.`)
	ErrNilReply            = errors.New(`Received a nil response.`)
	ErrNotConnected        = errors.New(`Client is not connected.`)
	ErrServerIsDown        = errors.New(`Server is down.`)
	ErrMissingCommand      = errors.New(`Missing command.`)
	ErrUnexpectedResponse  = errors.New(`Unexpected response from server.`)
	ErrExpectingPairs      = errors.New(`Expecting (field -> value) pairs.`)
	ErrNonBlockingRequired = errors.New(`This command requires a non-blocking connection.`)
	ErrMissingDestination  = errors.New(`Missing destination.`)
	ErrInvalidDestination  = errors.New(`Destination must be a pointer.`)
	ErrNotInitialized      = errors.New(`Client is not initialized. Forgot to use redis.New()?`)
)

// A redis client
type Client struct {
	redis *conn
}

func New() *Client {
	c := new(Client)
	return c
}

func (self *Client) Close() error {
	if self == nil {
		return ErrNotConnected
	}
	if self.redis != nil {
		self.redis.close()
		self.redis = nil
	}
	return nil
}

// Connects the client to the given host and port.
func (self *Client) Connect(host string, port uint) (err error) {

	if self == nil {
		return ErrNotInitialized
	}

	if self.redis != nil {
		self.Close()
	}

	connURL := fmt.Sprintf(`%s:%d`, host, port)

	if self.redis, err = dial(`tcp`, connURL); err != nil {
		return err
	}

	return nil
}

// ConnectWithTimeout attempts to connect to a redis-server, giving up after
// the specified time.
func (self *Client) ConnectWithTimeout(host string, port uint, timeout time.Duration) error {
	// TODO: Actually apply a timeout ;-).
	return self.Connect(host, port)
}

// ConnectUnixNonBlock attempts to create a non-blocking connection with an
// UNIX socket.
func (self *Client) ConnectUnixNonBlock(path string) error {
	return ErrNotConnected
}

// ConnectUnix attempts to create a connection with a UNIX socket.
func (self *Client) ConnectUnix(path string) error {
	return ErrNotConnected
}

// ConnectUnixWithTimeout attempts to create a connection with an UNIX socket,
// giving up after the specified time.
func (self *Client) ConnectUnixWithTimeout(path string, timeout time.Duration) error {
	return ErrNotConnected
}

// Command builds a command specified by the `values` interface and stores the
// result into the variable pointed by `dest`.
func (self *Client) Command(dest interface{}, values ...interface{}) error {
	if self == nil {
		return ErrNotInitialized
	}
	bvalues := make([][]byte, len(values))
	// Converting all input values into []byte.
	for i := range values {
		bvalues[i] = to.Bytes(values[i])
	}
	// Sending command to redis-server.
	return self.command(dest, bvalues...)
}

// Sendind a command and blocking until an answer is received.
func (self *Client) bcommand(c chan []string, values ...[]byte) error {
	return self.redis.blockCommand(c, values...)
}

func (self *Client) command(dest interface{}, values ...[]byte) error {
	return self.redis.syncCommand(dest, values...)
}
