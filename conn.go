package redis

import (
	"bufio"
	"errors"
	"github.com/xiam/resp"
	"log"
	"net"
	"strconv"
	"time"
)

var (
	ErrBadLine         = errors.New(`Bad line.`)
	ErrUnknownResponse = errors.New(`Unknown response.`)
)

var defaultTimeout = time.Second * 20

var endOfLine = []byte{'\r', '\n'}

const (
	respStringByte  = '+'
	respErrorByte   = '-'
	respIntegerByte = ':'
	respBulkByte    = '$'
	respArrayByte   = '*'
)

type conn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	timeout time.Duration
}

func dial(network string, address string) (*conn, error) {
	var nc net.Conn
	var err error

	if nc, err = net.Dial(network, address); err != nil {
		return nil, err
	}

	c := &conn{
		conn:    nc,
		timeout: defaultTimeout,
		reader:  bufio.NewReader(nc),
		writer:  bufio.NewWriter(nc),
	}

	return c, nil
}

func (self *conn) readBytes(data []byte) (err error) {
	if _, err := self.reader.Read(data); err != nil {
		return err
	}
	log.Printf("S: %#v\n", string(data))
	return nil
}

func (self *conn) readLine() ([]byte, error) {
	var data []byte
	var err error

	if data, err = self.reader.ReadSlice(endOfLine[1]); err != nil {
		return nil, err
	}

	log.Printf("S: %#v\n", string(data))

	return data, err
}

func (self *conn) read() ([]byte, error) {
	var head []byte
	var err error

	if head, err = self.reader.Peek(1); err != nil {
		return nil, err
	}

	switch head[0] {
	case respStringByte:
		return self.readLine()
	case respIntegerByte:
		return self.readLine()
	case respErrorByte:
		return self.readLine()
	case respArrayByte:
		var arr []byte
		var arrSize int

		if arr, err = self.readLine(); err != nil {
			return nil, err
		}

		if arrSize, err = strconv.Atoi(string(arr[1 : len(arr)-2])); err != nil {
			return nil, err
		}

		for i := 0; i < arrSize; i++ {
			var data []byte
			if data, err = self.read(); err != nil {
				return nil, err
			}
			arr = append(arr, data...)
		}

		log.Printf("S: %#v\n", string(arr))

		return arr, nil
	case respBulkByte:
		var bulk []byte
		var bulkSize int

		if bulk, err = self.readLine(); err != nil {
			return nil, err
		}

		if bulkSize, err = strconv.Atoi(string(bulk[1 : len(bulk)-2])); err != nil {
			return nil, err
		}

		if bulkSize > 0 {

			buf := make([]byte, bulkSize+2)

			if err = self.readBytes(buf); err != nil {
				return nil, err
			}

			buf = append(bulk, buf...)

			log.Printf("S: %#v\n", string(buf))

			return buf, nil
		}

		return bulk, nil
	}

	return nil, ErrUnknownResponse
}

func (self *conn) writeCommand(command ...interface{}) error {
	var data []byte
	var err error

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	return self.write(data)
}

func (self *conn) write(data []byte) error {
	var err error

	log.Printf("C: %#v\n", string(data))

	if _, err = self.writer.Write(data); err != nil {
		return err
	}

	return self.writer.Flush()
}

func (self *conn) close() error {
	return self.conn.Close()
}

func (self *conn) command(dest interface{}, command ...interface{}) error {
	var buf []byte
	var data []byte
	var err error

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	if err = self.write(data); err != nil {
		return err
	}

	if buf, err = self.read(); err != nil {
		return err
	}

	if dest == nil {
		return nil
	}

	return resp.Unmarshal(buf, dest)
}
