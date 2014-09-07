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

package redis

import (
	"errors"
	"fmt"
	"time"
)

const (
	// Default port for connecting to redis.
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

	if self.redis, err = dial(`tcp`, fmt.Sprintf(`%s:%d`, host, port)); err != nil {
		return err
	}

	return nil
}

// Connects to the given host and port, giving up after timeout.
func (self *Client) ConnectWithTimeout(host string, port uint, timeout time.Duration) error {
	// TODO: Actually apply timeout.
	return self.Connect(host, port)
}

// Creates a non-blocking connection between the client and the given UNIX
// socket.
func (self *Client) ConnectUnixNonBlock(path string) error {
	return ErrNotConnected
}

// Connects the client to the given UNIX socket.
func (self *Client) ConnectUnix(path string) error {
	return ErrNotConnected
}

// Connects the client to the given UNIX socket, giving up after timeout.
func (self *Client) ConnectUnixWithTimeout(path string, timeout time.Duration) error {
	return ErrNotConnected
}

// Sends a raw command and stores the response into a destination variable.
func (self *Client) Command(dest interface{}, values ...interface{}) error {
	if self == nil {
		return ErrNotInitialized
	}
	for i := range values {
		switch values[i].(type) {
		case []byte:
		default:
			values[i] = []byte(fmt.Sprintf("%v", values[i]))
		}
	}
	return self.redis.command(dest, values...)
}

func (self *Client) bcommand(c chan []string, values ...[]byte) error {
	if self == nil {
		return ErrNotInitialized
	}
	ivalues := make([]interface{}, len(values))
	for i := 0; i < len(values); i++ {
		ivalues[i] = values[i]
	}
	return self.redis.bcommand(c, ivalues...)
}

func (self *Client) command(dest interface{}, values ...[]byte) error {
	if self == nil {
		return ErrNotInitialized
	}
	ivalues := make([]interface{}, len(values))
	for i := 0; i < len(values); i++ {
		ivalues[i] = values[i]
	}
	return self.redis.command(dest, ivalues...)
}
