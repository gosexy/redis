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
	"bufio"
	"github.com/xiam/resp"
	"net"
	"sync"
	"time"
)

type conn struct {
	conn         net.Conn
	subscription bool
	timeout      time.Duration
	mu           *sync.Mutex
	reader       *bufio.Reader
	writer       *bufio.Writer
	decoder      *resp.Decoder
	encoder      *resp.Encoder
}

func newConn(nc net.Conn) (*conn, error) {
	c := &conn{
		conn:    nc,
		timeout: defaultTimeout,
		mu:      &sync.Mutex{},
	}

	c.reader = bufio.NewReader(c.conn)
	c.writer = bufio.NewWriter(c.conn)

	c.decoder = resp.NewDecoder(c.reader)
	c.encoder = resp.NewEncoder(c.writer)

	return c, nil
}

func dial(network string, address string) (*conn, error) {
	var nc net.Conn
	var err error

	if nc, err = net.Dial(network, address); err != nil {
		return nil, err
	}

	return newConn(nc)
}

func dialTimeout(network string, address string, timeout time.Duration) (*conn, error) {
	var nc net.Conn
	var err error

	if nc, err = net.DialTimeout(network, address, timeout); err != nil {
		return nil, err
	}

	return newConn(nc)
}

func (c *conn) close() error {
	c.subscription = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	return nil
}

func (c *conn) syncCommand(dest interface{}, command ...[]byte) (err error) {
	if c.conn == nil {
		return ErrNotConnected
	}

	if err = c.encoder.Encode(command); err != nil {
		return err
	}

	// This is an atomic operation, for now.
	c.mu.Lock()

	if err = c.writer.Flush(); err != nil {
		c.mu.Unlock()
		return err
	}

	err = c.decoder.Decode(dest)

	c.mu.Unlock()

	if dest == nil && err == resp.ErrExpectingDestination {
		return nil
	}

	if err == resp.ErrMessageIsNil {
		return ErrNilReply
	}

	return err
}

func (c *conn) asyncCommand(dest interface{}, command ...[]byte) (chan error, error) {
	var errProm chan error

	errProm = make(chan error)

	go func() {
		// TODO: make it actually asynchronous.
		err := c.syncCommand(dest, command...)
		errProm <- err
	}()

	return errProm, nil
}

func (c *conn) blockCommand(d chan []string, command ...[]byte) error {
	var err error

	if err = c.encoder.Encode(command); err != nil {
		return err
	}

	c.mu.Lock()

	if err = c.writer.Flush(); err != nil {
		return err
	}

	c.mu.Unlock()

	c.subscription = true

	if d != nil {
		go func(d chan []string) {
			for c.subscription {
				var s []string
				err = c.decoder.Decode(&s)
				if err != nil {
					// panic(err.Error())
					return
				}
				d <- s
			}
		}(d)
	}

	return err
}
