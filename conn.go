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
	"bytes"
	"github.com/xiam/resp"
	"net"
	"strconv"
	"sync"
	"time"
)

var endOfLine = []byte{'\r', '\n'}

const (
	respStringByte  = '+'
	respErrorByte   = '-'
	respIntegerByte = ':'
	respBulkByte    = '$'
	respArrayByte   = '*'
)

type conn struct {
	conn         net.Conn
	subscription bool
	timeout      time.Duration
	mu           *sync.Mutex
	reader       *bufio.Reader
	writer       *bufio.Writer
}

func newConn(nc net.Conn) (*conn, error) {
	c := &conn{
		conn:    nc,
		timeout: defaultTimeout,
		mu:      &sync.Mutex{},
	}

	c.reader = bufio.NewReader(c.conn)
	c.writer = bufio.NewWriter(c.conn)

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

func (c *conn) read() (buf []byte, err error) {
	var head []byte

	// Attempts to read message type.
	if head, err = c.reader.Peek(1); err != nil {
		return nil, err
	}

	switch head[0] {
	case respStringByte:
		return c.readNextLine()
	case respIntegerByte:
		return c.readNextLine()
	case respErrorByte:
		return c.readNextLine()
	case respArrayByte:
		var size int
		var buf []byte

		if buf, err = c.readNextLine(); err != nil {
			return nil, err
		}

		if size, err = strconv.Atoi(string(buf[1 : len(buf)-2])); err != nil {
			return nil, err
		}

		if size < 0 {
			return c.readNextLine()
		}

		var data []byte
		for i := 0; i < size; i++ {
			if data, err = c.read(); err != nil {
				return nil, err
			}
			buf = append(buf, data...)
		}

		return buf, nil
	case respBulkByte:
		var buf []byte
		var data []byte
		var size int

		if buf, err = c.peekNextLine(); err != nil {
			return nil, err
		}

		if size, err = strconv.Atoi(string(buf[1 : len(buf)-2])); err != nil {
			return nil, err
		}

		// Nil
		if size < 1 {
			return c.readNextLine()
		}

		size = len(buf) + size + 2

		if size > 0 {
			var n int
			data = make([]byte, 0, size)

			for size > 0 {
				chunk := make([]byte, size)
				if n, err = c.reader.Read(chunk); err != nil {
					return nil, err
				}
				data = append(data, chunk[:n]...)
				size = size - n
			}
		}

		return data, nil
	}

	return nil, ErrUnknownResponse
}

func (c *conn) write(message []byte) (err error) {
	_, err = c.writer.Write(message)
	c.writer.Flush()
	return
}

func (c *conn) peekNextLine() (buf []byte, err error) {
	var n int

	b := c.reader.Buffered()
	if b > 100 {
		b = 100
	}

	if buf, err = c.reader.Peek(b); err != nil {
		return nil, err
	}

	if n = bytes.IndexByte(buf, endOfLine[1]); n < 0 {
		return nil, ErrNotEnoughData
	}

	return buf[:n+1], nil
}

func (c *conn) readNextLine() (data []byte, err error) {
	if data, err = c.reader.ReadBytes(endOfLine[1]); err != nil {
		return nil, err
	}

	return data, err
}

func (c *conn) writeCommand(command ...interface{}) error {
	var data []byte
	var err error

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	return c.write(data)
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
	var data []byte
	var buf []byte

	if c.conn == nil {
		return ErrNotConnected
	}

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	// This is an atomic operation, for now.
	c.mu.Lock()

	if err = c.write(data); err != nil {
		return err
	}

	if buf, err = c.read(); err != nil {
		return err
	}

	c.mu.Unlock()

	if dest == nil {
		return nil
	}

	if len(buf) == 0 {
		return ErrNilReply
	}

	err = resp.Unmarshal(buf, dest)

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
	var data []byte
	var err error

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	c.mu.Lock()

	if err = c.write(data); err != nil {
		return err
	}

	c.mu.Unlock()

	c.subscription = true

	if d != nil {
		go func(d chan []string) {
			var buf []byte
			for c.subscription {
				var s []string
				buf, err = c.read()
				if err != nil {
					return
				}
				err = resp.Unmarshal(buf, &s)
				d <- s
			}
		}(d)
	}

	return err
}
