package redis

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/xiam/resp"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	ErrBadLine         = errors.New(`Bad line.`)
	ErrUnknownResponse = errors.New(`Unknown response.`)
	ErrNotEnoughData   = errors.New(`Not enough data.`)
)

const (
	readBufferSize    = 4000
	channelBufferSize = 128
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
	conn          net.Conn
	writeBuf      chan []byte
	readBuf       chan []byte
	shouldWrite   chan bool
	shouldRead    chan bool
	writer        *bytes.Buffer
	readerBuf     *bytes.Buffer
	quitReading   chan bool
	quitWriting   chan bool
	quit          chan bool
	reader        *bufio.Reader
	timeout       time.Duration
	commandMutex  *sync.Mutex
	readerMutex   *sync.Mutex
	writerMutex   *sync.Mutex
	readLineMutex *sync.Mutex
	lineBuf       []chan []byte
}

func dial(network string, address string) (*conn, error) {
	var nc net.Conn
	var err error

	if nc, err = net.Dial(network, address); err != nil {
		return nil, err
	}

	c := &conn{
		conn:          nc,
		timeout:       defaultTimeout,
		writer:        bytes.NewBuffer(nil),
		readerBuf:     bytes.NewBuffer(nil),
		readerMutex:   new(sync.Mutex),
		commandMutex:  new(sync.Mutex),
		writerMutex:   new(sync.Mutex),
		readLineMutex: new(sync.Mutex),
		quitReading:   make(chan bool),
		quitWriting:   make(chan bool),
		quit:          make(chan bool),
		shouldWrite:   make(chan bool, channelBufferSize),
		shouldRead:    make(chan bool, channelBufferSize),
		writeBuf:      make(chan []byte, channelBufferSize),
		readBuf:       make(chan []byte, channelBufferSize),
		lineBuf:       make([]chan []byte, 0, channelBufferSize),
	}

	c.reader = bufio.NewReader(c.readerBuf)

	go c.asyncStatus()

	go c.asyncConnReader()

	go c.asyncConnWriter()

	go c.asyncReadWriter()

	return c, nil
}

func (self *conn) asyncStatus() {
	for {
		select {
		case <-self.quit:
			if self.conn != nil {
				log.Println("request to quit.")
				self.quitWriting <- true
				log.Println("CLOSED 1")
				self.quitReading <- true
				log.Println("CLOSED 2")
				close(self.quit)
				close(self.quitWriting)
				close(self.quitReading)
				close(self.shouldRead)
				close(self.shouldWrite)
				close(self.writeBuf)
				close(self.readBuf)
				log.Println("CLOSED 3")
			}
			return
		}
	}
}

func (self *conn) writeNextLine(data []byte) error {
	self.writeBuf <- data
	return nil
}

func (self *conn) reserveNextLine() chan []byte {
	nextLine := make(chan []byte)
	self.lineBuf = append(self.lineBuf, nextLine)
	return nextLine
}

func (self *conn) asyncConnReader() {
	var err error
	var data []byte
	var n int

	for {
		data = make([]byte, readBufferSize)
		if n, err = self.conn.Read(data); err != nil {
			if err == io.EOF {
				self.close()
			}
			log.Printf("read error(1): %q\n", err)
			return
		}
		if n > 0 {
			//log.Printf("n: %d: %#v\n", n, string(data[:n]))

			self.readerBuf.Write(data[:n])
			self.shouldRead <- true
		}
	}

}

func (self *conn) asyncReadWriter() {
	var err error
	var next []byte

	for {
		select {
		case <-self.quitReading:
			return
		case <-self.shouldRead:
			//self.readerMutex.Lock()
			for {
				if self.readerBuf.Len()+self.reader.Buffered() > 0 {
					if next, err = self.readNextMessage(); err != nil {
						log.Printf("next message error(2): %q\n", err)
						break
					}
					if len(self.lineBuf) > 0 {
						self.lineBuf[0] <- next
						self.lineBuf = self.lineBuf[1:]
					} else {
						//log.Printf("no buff\n")
					}
					//log.Printf("NEXT: %#v\n", string(next))
					//self.readBuf <- next
				} else {
					//log.Println("ignored read\n")
					break
				}
			}
			//self.readerMutex.Unlock()
		case <-self.shouldWrite:
			if self.writer.Len() > 0 {
				self.writerMutex.Lock()
				if err = self.write(self.writer.Bytes()); err != nil {
					//log.Printf("write error: %q\n", err)
					continue
				}
				//log.Printf("written %d messages.\n", self.writer.Len())
				self.writer.Reset()
				self.writerMutex.Unlock()
				//log.Printf("write ok\n")
			} else {
				//log.Printf("ignored message\n")
			}
		}
	}

}

func (self *conn) asyncConnWriter() {
	var message []byte

	for {
		select {
		case <-self.quitWriting:
			return
		case message = <-self.writeBuf:
			self.writerMutex.Lock()
			self.writer.Write(message)
			self.writerMutex.Unlock()
			self.shouldWrite <- true
		}
	}
}

func (self *conn) peekNextLine() (buf []byte, err error) {
	var n int

	//log.Printf("peekNextLine\n")

	b := self.readerBuf.Len() + self.reader.Buffered()
	if b > 100 {
		b = 100
	}
	//log.Printf("peeking :%d\n", b)

	if buf, err = self.reader.Peek(b); err != nil {
		//log.Printf("peek error: %q\n", err)
		return nil, err
	}

	if n = bytes.IndexByte(buf, endOfLine[1]); n < 0 {
		return nil, ErrNotEnoughData
	}

	//log.Printf("peekd: %#v\n", string(buf[:n+1]))

	return buf[:n+1], nil
}

func (self *conn) readNextLine() (data []byte, err error) {
	self.readLineMutex.Lock()
	defer self.readLineMutex.Unlock()

	if data, err = self.reader.ReadBytes(endOfLine[1]); err != nil {
		log.Printf("last line err: %q\n", err)
		return nil, err
	}

	log.Printf("L: %#v\n", string(data))
	//log.Printf("slice ok")

	return data, err
}

func (self *conn) readNextMessage() ([]byte, error) {
	var head []byte
	var err error

	// log.Println("readNExtMessage")

	/*
		self.readerMutex.Lock()

		defer func() {
			self.readerMutex.Unlock()
		}()
	*/

	// Attempts to read message type.
	if head, err = self.reader.Peek(1); err != nil {
		return nil, err
	}

	log.Printf("peek: %#v\n", string(head))

	switch head[0] {
	case respStringByte:
		//log.Println("STRING")
		return self.readNextLine()
	case respIntegerByte:
		//log.Println("INTEGER")
		return self.readNextLine()
	case respErrorByte:
		//log.Println("BYTE")
		return self.readNextLine()
	case respArrayByte:
		//log.Println("ARRAY")
		var size int
		var buf []byte

		if buf, err = self.readNextLine(); err != nil {
			return nil, err
		}

		if size, err = strconv.Atoi(string(buf[1 : len(buf)-2])); err != nil {
			return nil, err
		}

		if size < 0 {
			return self.readNextLine()
		}

		var data []byte
		for i := 0; i < size; i++ {
			if data, err = self.readNextMessage(); err != nil {
				log.Printf("exitr\n")
				return nil, err
			}
			log.Printf("OK ARRAY\n")
			buf = append(buf, data...)
		}

		log.Printf("S(1): %#v\n", string(buf))

		return buf, nil
	case respBulkByte:
		var buf []byte
		var data []byte
		var size int

		log.Println("BULK")

		if buf, err = self.peekNextLine(); err != nil {
			return nil, err
		}

		if size, err = strconv.Atoi(string(buf[1 : len(buf)-2])); err != nil {
			return nil, err
		}

		log.Printf("size: %d\n", size)

		// Nil
		if size < 1 {
			return self.readNextLine()
		}

		size = len(buf) + size + 2

		if self.reader.Buffered()+self.readerBuf.Len() < size {
			log.Println("BULK: not enough data.")
			log.Printf("got: %d, expecting: %d\n", self.reader.Buffered(), size)
			log.Printf("buf: %d\n", self.readerBuf.Len())
			//log.Printf("buf: %#v\n", string(buf))
			return buf, ErrNotEnoughData
		}

		if size > 0 {
			var n int
			data = make([]byte, 0, size)

			for size > 0 {
				chunk := make([]byte, size)
				if n, err = self.reader.Read(chunk); err != nil {
					return nil, err
				}
				data = append(data, chunk[:n]...)
				size = size - n
				log.Printf("remaining: %d, n: %d\n", size, n)
			}

			//data = append(buf, data...)

			log.Printf("S(2): %#v\n", string(data))
		}

		log.Printf("S: %#v\n", string(data))

		return data, nil
	}

	return nil, ErrUnknownResponse
}

func (self *conn) writeCommand(command ...interface{}) error {
	var data []byte
	var err error

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	return self.writeNextLine(data)
}

func (self *conn) write(data []byte) error {
	var err error

	log.Printf("C: %#v\n", string(data))

	if _, err = self.conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (self *conn) close() error {
	if self.conn != nil {
		self.conn.Close()
		self.quit <- true
		self.conn = nil
	}
	return nil
}

func (self *conn) asyncCommand(dest interface{}, command ...interface{}) (chan error, error) {
	var buf []byte
	var errProm chan error
	var err error

	errProm = make(chan error)

	go func() {
		var data []byte

		if data, err = resp.Marshal(command); err != nil {
			errProm <- err
			return
		}

		var nextLine chan []byte

		self.commandMutex.Lock()

		//log.Printf("Q: %#v\n", string(data))

		self.writeNextLine(data)
		nextLine = self.reserveNextLine()

		self.commandMutex.Unlock()

		buf = <-nextLine

		if dest == nil {
			errProm <- nil
			return
		}

		errProm <- resp.Unmarshal(buf, dest)
	}()

	return errProm, nil
}

func (self *conn) read() ([]byte, error) {
	var nextLine chan []byte
	var buf []byte
	nextLine = self.reserveNextLine()
	buf = <-nextLine
	return buf, nil
}

func (self *conn) command(dest interface{}, command ...interface{}) error {
	var data []byte
	var buf []byte
	var err error

	if self.conn == nil {
		return ErrNotConnected
	}

	if data, err = resp.Marshal(command); err != nil {
		return err
	}

	var nextLine chan []byte

	self.commandMutex.Lock()

	log.Printf("Q: %#v\n", string(data))

	self.writeNextLine(data)
	nextLine = self.reserveNextLine()

	self.commandMutex.Unlock()

	buf = <-nextLine

	if dest == nil || len(buf) == 0 {
		return ErrNilReply
	}

	err = resp.Unmarshal(buf, dest)

	if err == resp.ErrMessageIsNil {
		return ErrNilReply
	}

	return err
}
