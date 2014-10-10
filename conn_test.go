package redis

import (
	"bytes"
	"errors"
	"log"
	"strconv"
	"sync"
	"testing"
)

var (
	errTestFailed = errors.New(`Test failed.`)
)

func TestNoCGOConn(t *testing.T) {
	var c *conn
	var err error

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	if err = c.close(); err != nil {
		t.Fatal(err)
	}
}

func TestNoCGOPing(t *testing.T) {
	var c *conn
	var err error
	var data []byte

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	if err = c.writeCommand([]byte("PING")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte("+PONG\r\n")) == false {
		t.Fatal()
	}

	if err = c.close(); err != nil {
		t.Fatal(err)
	}

}

func TestNoCGODelete(t *testing.T) {
	var c *conn
	var err error

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	if err = c.writeCommand([]byte("DEL"), []byte("testKey")); err != nil {
		t.Fatal(err)
	}

	if _, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if err = c.close(); err != nil {
		t.Fatal(err)
	}
}

func TestNoCGOSetGet(t *testing.T) {
	var c *conn
	var err error
	var data []byte

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	defer c.close()

	if err = c.writeCommand([]byte("SET"), []byte("testKey"), []byte("Hello ☀!")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte("+OK\r\n")) == false {
		t.Fatal()
	}

	if err = c.writeCommand([]byte("GET"), []byte("testKey")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte("$10\r\nHello ☀!\r\n")) == false {
		t.Fatal()
	}

	if err = c.writeCommand([]byte("SET"), []byte("testKey"), []byte("Hello\r\n☀!")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte("+OK\r\n")) == false {
		t.Fatal()
	}

	if err = c.writeCommand([]byte("GET"), []byte("testKey")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte("$11\r\nHello\r\n☀!\r\n")) == false {
		t.Fatal()
	}

}

func TestNoCGOPushList(t *testing.T) {
	var c *conn
	var err error
	var data []byte

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	defer c.close()

	if err = c.writeCommand([]byte("DEL"), []byte("testKey")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte(":1\r\n")) == false {
		t.Fatal()
	}

	if err = c.writeCommand([]byte("RPUSH"), []byte("testKey"), []byte("Hello ☀!")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte(":1\r\n")) == false {
		t.Fatal()
	}

	if err = c.writeCommand([]byte("RPUSH"), []byte("testKey"), []byte("How are ★?")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte(":2\r\n")) == false {
		t.Fatal()
	}

	if err = c.writeCommand([]byte("LRANGE"), []byte("testKey"), []byte("0"), []byte("-1")); err != nil {
		t.Fatal(err)
	}

	if data, err = c.read(); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(data, []byte("*2\r\n$10\r\nHello ☀!\r\n$12\r\nHow are ★?\r\n")) == false {
		t.Fatal()
	}
}

func TestNoCGOPingCommand(t *testing.T) {
	var dest string
	var err error
	var c *conn

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	defer c.close()

	if err = c.syncCommand(&dest, []byte("PING")); err != nil {
		t.Fatal(err)
	}

	if dest != "PONG" {
		t.Fatal()
	}
}

func TestNoCGODelSetGetCommand(t *testing.T) {
	var err error
	var s string
	var c *conn

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	defer c.close()

	if err = c.syncCommand(nil, []byte("DEL"), []byte("testKey")); err != nil {
		if err != ErrNilReply {
			t.Fatal(err)
		}
	}

	if err = c.syncCommand(&s, []byte("SET"), []byte("testKey"), []byte("Foo Bar!")); err != nil {
		t.Fatal(err)
	}

	if s != "OK" {
		t.Fatal()
	}

	if err = c.syncCommand(&s, []byte("GET"), []byte("testKey")); err != nil {
		t.Fatal(err)
	}

	if s != "Foo Bar!" {
		t.Fatal()
	}

}

func TestNoCGOAsyncCommand(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	var c *conn

	if c, err = dial(testProto, testAddress); err != nil {
		t.Fatal(err)
	}

	defer c.close()

	l := 1000

	results := make(map[int]bool, l)

	for i := 0; i < l; i++ {
		wg.Add(1)

		results[i] = false

		go func(i int) {
			var a string
			var e string
			var errProm chan error
			var err error

			e = strconv.Itoa(i)
			if errProm, err = c.asyncCommand(&a, []byte("ECHO"), []byte(e)); err != nil {
				log.Fatalf("async: %q\n", err)
			}

			err = <-errProm

			if err != nil {
				log.Fatalf("async(2): %q\n", err)
			}

			if a != e {
				log.Fatalf("%d: got %s, expecting %s.", i, a, e)
			}

			results[i] = true

			wg.Done()
		}(i)
	}

	wg.Wait()

	for i := 0; i < l; i++ {
		if results[i] == false {
			log.Fatalf("Missing result for index %d.", i)
		}
	}

}
