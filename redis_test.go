package redis

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testHost    string
	testAddress string
	testPort    uint
)

const (
	testProto = `tcp`
)

func init() {
	// Getting host and port from command line.
	host := flag.String("host", "10.1.2.201", "Test hostname or address.")
	port := flag.Uint("port", 6379, "Port.")

	flag.Parse()

	testHost = *host
	testPort = *port

	testAddress = fmt.Sprintf("%s:%d", testHost, testPort)

	log.Printf("Running tests against host %s:%d.\n", testHost, testPort)
}

func TestConnect(t *testing.T) {
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	if err = client.Close(); err != nil {
		t.Fatalf("Client should be able to close the connection.")
	}

	if err = client.Connect(testHost, 0); err == nil {
		t.Fatalf("Client should not be able to connect to a unused port.")
	}

	if err = client.Close(); err != nil {
		t.Fatalf("Client should be able to close the connection.")
	}
}

func TestPing(t *testing.T) {
	var s string
	var err error
	var client *Client

	client = New()

	if err = client.ConnectWithTimeout(testHost, testPort, 0); err != nil {
		t.Fatalf("Client failed to connect and set a timeout: %q", err)
	}

	defer client.Close()

	if s, err = client.Ping(); err != nil {
		t.Fatalf("Command PING failed: %q", err)
	}

	if s != "PONG" {
		t.Fatal("Expecting PONG reply.")
	}

	client.Quit()
}

func TestSimpleSet(t *testing.T) {
	var s string
	var b bool
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	if s, err = client.Set("foo", "hello world."); err != nil {
		t.Fatalf("Command SET failed: %q", err)
	}

	if s != "OK" {
		t.Fatal("Expecting OK response.")
	}

	if b, err = client.SetNX("foo", "exists!"); err != nil {
		t.Fatalf("Command SETNX failed: %q", err)
	}

	if b == true {
		t.Fatal()
	}
}

func TestGet(t *testing.T) {
	var s string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal("Client failed to connect: ", err)
	}

	defer client.Close()

	// Setting the FOO key.
	client.Set("foo", "hello")

	// Getting the FOO key.
	if s, err = client.Get("foo"); err != nil {
		t.Fatalf("Command GET failed: %q", err)
	}

	if s != "hello" {
		t.Fatalf("Could not SET/GET value.")
	}

	// Deleting key.
	client.Del("foo")

	// Attempting to retrieve deleted key.
	s, err = client.Get("foo")

	if s != "" {
		t.Fatalf("Expecting an empty string.")
	}

	if err != ErrNilReply {
		t.Fatalf("Expecting a redis.ErrNilReply error, got %q.", err)
	}

	// Making sure https://github.com/gosexy/redis/issues/23 does not interfere
	// with https://github.com/gosexy/redis/issues/12.
	client.Del("test")

	client.HMSet("test", "123", "*", "456", "*", "789", "*")

	var vals []string
	vals, err = client.HMGet("test", "1232", "456")

	if err != nil {
		t.Fatal(err)
	}

	if vals[0] != "" || vals[1] != "*" {
		t.Fatalf("Unexpected values.")
	}

}

func TestSetGetUnicode(t *testing.T) {
	var r string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	if r, err = client.Set("heart", "♥"); err != nil {
		t.Fatalf("Command failed: %q", err)
	}

	if r != "OK" {
		t.Fatalf("Could not SET binary value.")
	}

	if r, err = client.Get("heart"); err != nil {
		t.Fatalf("Command failed: %q", err)
	}

	if r != "♥" {
		t.Fatalf("Could not GET binary value.")
	}
}

func TestDel(t *testing.T) {
	var i int64
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	client.Set("counter", 0)

	i, err = client.Del("counter")

	if err != nil {
		t.Fatal(err)
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	i, err = client.Del("counter")

	if err != nil {
		t.Fatal(err)
	}

	if i != 0 {
		t.Fatalf("Failed")
	}
}

func TestList(t *testing.T) {
	var items []string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	_, err = client.Del("list")

	if err != nil {
		t.Fatalf("Command failed: %q", err)
	}

	for i := 0; i < 10; i++ {
		_, err = client.LPush("list", fmt.Sprintf("element-%d", i))
		if err != nil {
			t.Fatalf("Command failed: %s", err.Error())
		}
	}

	items, err = client.LRange("list", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(items) != 10 {
		t.Fatalf("Failed.")
	}

}

func TestAppendGetRange(t *testing.T) {
	var err error
	var s string
	var b bool
	var i int64
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	_, err = client.Del("mykey")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	b, err = client.Exists("mykey")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if b == true {
		t.Fatalf("Failed.")
	}

	i, err = client.Append("mykey", "Hello")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 5 {
		t.Fatalf("Failed.")
	}

	i, err = client.Append("mykey", " World")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 11 {
		t.Fatalf("Failed.")
	}

	s, err = client.Get("mykey")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "Hello World" {
		t.Fatalf("Failed.")
	}

	_, err = client.Del("ts")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	_, err = client.Append("ts", "0043")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	_, err = client.Append("ts", "0035")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	s, err = client.GetRange("ts", 0, 3)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "0043" {
		t.Fatalf("Failed.")
	}

	s, err = client.GetRange("ts", 4, 7)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "0035" {
		t.Fatalf("Failed.")
	}
}

func TestBitCount(t *testing.T) {
	var err error
	var i int64
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	_, err = client.Set("mykey", "foobar")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	i, err = client.BitCount("mykey")

	if i != 26 {
		t.Fatalf("Failed.")
	}

	i, err = client.BitCount("mykey", 0, 0)

	if i != 4 {
		t.Fatalf("Failed.")
	}

	i, err = client.BitCount("mykey", 1, 1)

	if i != 6 {
		t.Fatalf("Failed.")
	}

}

func TestIncrDecr(t *testing.T) {
	var i int64
	var err error
	var s string
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	// -> 0
	client.Set("counter", 0)

	// -> 1
	client.Incr("counter")

	err = client.Command(&i, "GET", "counter")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// -> 1 + 10
	client.IncrBy("counter", 10)

	s, err = client.Get("counter")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "11" {
		t.Fatalf("Failed")
	}

	// -> 11 - 1
	i, err = client.Decr("counter")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	// 10 - 6
	i, err = client.DecrBy("counter", 6)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 4 {
		t.Fatalf("Failed")
	}

	s, err = client.IncrByFloat("counter", 0.01)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "4.01" {
		t.Fatalf("Failed")
	}

	s, err = client.IncrByFloat("counter", -0.02)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "3.99" {
		t.Fatalf("Failed")
	}
}

func TestDump(t *testing.T) {
	var s string
	var d string
	var ls []string
	var i int64
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	// Deleting key
	client.Del("mykey")

	// Trying to RPUSHX, but key does not exists.
	i, err = client.RPushX("mykey", 3)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 0 {
		t.Fatalf("Failed")
	}

	// Pushing 1, 2
	i, err = client.RPush("mykey", 1, 2)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Pushing 3
	i, err = client.RPushX("mykey", 3)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 3 {
		t.Fatalf("Failed")
	}

	// Generating a dump
	d, err = client.Dump("mykey")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	// Deleting key
	client.Del("mykey")

	// Restoring key from dump.
	s, err = client.Restore("mykey", 0, d)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Key type
	s, err = client.Type("mykey")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if s != "list" {
		t.Fatalf("Failed")
	}

	// Getting values.
	ls, err = client.LRange("mykey", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 3 {
		t.Fatalf("Failed")
	}

}

func TestExpire(t *testing.T) {
	var b bool
	var i int64
	var s string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Setting key
	client.Set("mykey", "Hello")

	// Setting expiration
	b, err = client.Expire("mykey", 10)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Checking time to live
	i, err = client.TTL("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 10 {
		t.Fatalf("Failed")
	}

	// Making key persistent
	b, err = client.Persist("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Checking time to live
	i, err = client.TTL("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != -1 {
		t.Fatalf("Failed")
	}

	// Checking if the key exists
	b, err = client.Exists("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Setting past expiration
	b, err = client.ExpireAt("mykey", 1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Checking if the key exists
	b, err = client.Exists("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != false {
		t.Fatalf("Failed")
	}

	// Creating key
	client.Set("mykey", "Hello")

	// Setting expiration (ms)
	b, err = client.PExpire("mykey", 1500)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == false {
		t.Fatalf("Failed")
	}

	// Checking time to live
	i, err = client.TTL("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i < 1 {
		t.Fatalf("Failed")
	}

	// Checking time to live (ms)
	i, err = client.PTTL("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i < 1000 {
		t.Fatalf("Failed")
	}

	// Setting past expiration
	b, err = client.PExpireAt("mykey", 1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Checking if the key exists
	b, err = client.Exists("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == true {
		t.Fatalf("Failed")
	}

	// Deleting key
	client.Del("mykey")

	// Setting value with expiration
	s, err = client.PSetEx("mykey", 1000, "Hello")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Confirming TTL
	i, err = client.PTTL("mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i < 100 {
		t.Fatalf("Failed")
	}

}

func TestSet(t *testing.T) {
	var i int64
	var b bool
	var s string
	var ls []string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Deleting
	client.Del("myset")

	// Adding
	i, err = client.SAdd("myset", "Hello", "World", "World")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Counting elements
	i, err = client.SCard("myset")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Deleting
	client.Del("key1", "key2")

	// Adding
	client.SAdd("key1", "a", "b", "c")
	client.SAdd("key2", "c", "d", "e")

	// Difference
	ls, err = client.SDiff("key1", "key2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed")
	}

	// Difference
	i, err = client.SDiffStore("key3", "key1", "key2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Intersection
	ls, err = client.SInter("key1", "key2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 1 {
		t.Fatalf("Failed")
	}

	// Intersection
	i, err = client.SInterStore("key3", "key1", "key2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Is member?
	b, err = client.SIsMember("key3", "c")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Members
	ls, err = client.SMembers("key1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 3 {
		t.Fatalf("Failed")
	}

	// Move
	b, err = client.SMove("key1", "key2", "a")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b != true {
		t.Fatalf("Failed")
	}

	// Pop
	s, err = client.SPop("key1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s == "" {
		t.Fatalf("Failed")
	}

	// Random members
	ls, err = client.SRandMember("key2", 2)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed")
	}

	// Random members
	i, err = client.SRem("key2", "c")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Union
	ls, err = client.SUnion("key1", "key2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 4 {
		t.Fatalf("Failed")
	}

	// Union
	i, err = client.SUnionStore("key3", "key1", "key2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 4 {
		t.Fatalf("Failed")
	}

}

func TestZSet(t *testing.T) {
	var i int64
	var s string
	var ls []string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Deleting
	client.Del("myset")

	// Adding
	i, err = client.ZAdd("myset", 1, "one", 2, "teo")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Counting elements
	i, err = client.ZCard("myset")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Counting elements
	i, err = client.ZCount("myset", "-inf", "+inf")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Increment
	s, err = client.ZIncrBy("myset", 1, "one")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "2" {
		t.Fatalf("Failed")
	}

	// Deleting
	client.Del("zset1", "zset2")

	// Adding
	client.ZAdd("zset1", 1, "one", 2, "two")
	client.ZAdd("zset2", 1, "one", 2, "two", 3, "three")

	// Intersection
	i, err = client.ZInterStore("out", 2, "zset1", "zset2", "WEIGHTS", 2, 3)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Range
	ls, err = client.ZRange("out", 0, -1, "WITHSCORES")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 4 {
		t.Fatalf("Failed")
	}

	// Adding
	client.Del("myzset")
	client.ZAdd("myzset", 1, "one", 2, "two", 3, "three")

	// Range
	ls, err = client.ZRangeByScore("myzset", 1, 2)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed")
	}

	// Rank
	i, err = client.ZRank("myzset", "two")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Reverse Rank
	i, err = client.ZRevRank("myzset", "one")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Reverse Rank
	i, err = client.ZRevRank("myzset", "none")

	if i != 0 {
		t.Fatalf("Expecting error.")
	}

	// Remove
	i, err = client.ZRem("myzset", "two", "three")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Remove
	i, err = client.ZRemRangeByRank("myzset", 0, 1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Adding
	client.Del("myzset")
	client.ZAdd("myzset", 1, "one", 2, "two", 3, "three")

	// Remove
	i, err = client.ZRemRangeByScore("myzset", "-inf", "(2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Range
	ls, err = client.ZRevRange("myzset", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed")
	}

	// Range
	ls, err = client.ZRevRangeByScore("myzset", "+inf", "-inf")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed")
	}

	// Score
	i, err = client.ZScore("myzset", "two")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// Deleting
	client.Del("zset1", "zset2")

	// Adding
	client.ZAdd("zset1", 1, "one", 2, "two")
	client.ZAdd("zset2", 1, "one", 2, "two", 3, "three")

	// Intersection
	i, err = client.ZUnionStore("out", 2, "zset1", "zset2", "WEIGHTS", 2, 3)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 3 {
		t.Fatalf("Failed")
	}

}

func TestSubscribe(t *testing.T) {
	var ls []string
	var err error

	ok := make(chan bool)

	consumer := New()

	if err = consumer.Connect(testHost, testPort); err != nil {
		t.Fatal("Failed to connect: ", err)
	}

	rec := make(chan []string)

	go func() {
		select {
		case ls = <-rec:
			t.Logf("Got: %v\n", ls)
			ok <- true
			return
		}
	}()

	if err = consumer.Subscribe(rec, "channel"); err != nil {
		t.Fatal("Failed to subscribe: ", err)
	}

	<-ok

	consumer.Unsubscribe("channel")

	consumer.Quit()
}

func TestPublishAndSubscribe(t *testing.T) {
	var err error
	var received map[int]bool
	var tests int

	wg := new(sync.WaitGroup)

	mu := new(sync.Mutex)

	tests = 10

	// Subscriber
	consumer := New()

	if err = consumer.Connect(testHost, testPort); err != nil {
		t.Fatal("Failed to connect: ", err)
	}

	rec := make(chan []string, tests)

	subscribed := make(chan bool)

	go func() {
		received = make(map[int]bool)
		for {
			// Receiving messages.
			select {
			case ls := <-rec:
				if ls[0] == `message` {
					i, _ := strconv.Atoi(ls[2])
					received[i] = true
					// Marking job as done.
					wg.Done()
				} else if ls[0] == `subscribe` {
					subscribed <- true
				} else if ls[0] == `unsubscribe` {
					return
				}
			}
		}
	}()

	if err = consumer.Subscribe(rec, "channel"); err != nil {
		t.Fatal("Subscribe() ", err)
	}

	// Waiting for subscription to happen.
	<-subscribed

	// Publisher
	publisher := New()

	if err = publisher.Connect(testHost, testPort); err != nil {
		t.Fatal("Failed to connect: ", err)
	}

	// Publishing
	var clients int64
	for i := 0; i < tests; i++ {
		mu.Lock()
		wg.Add(1)
		clients, err = publisher.Publish("channel", i)
		if err != nil {
			t.Fatal("Publish(): ", err)
		}
		if clients < 1 {
			t.Fatal("We should have at least one subscribed client.")
		}
		mu.Unlock()
	}

	// Waiting jobs to finish.
	wg.Wait()

	if _, err = publisher.Quit(); err != nil {
		t.Fatal("Quit(): ", err)
	}

	if err = consumer.Unsubscribe("channel"); err != nil {
		t.Fatal("Unsubscribe(): ", err)
	}

	if _, err = consumer.Quit(); err != nil {
		t.Fatal("Quit(): ", err)
	}

	// Verifying data map.
	for i := 0; i < tests; i++ {
		if received[i] == false {
			t.Fatal(`The "test" map should be populated with true values.`)
		}
	}
}

func TestPSubscribe(t *testing.T) {
	var ls []string
	var err error

	ok := make(chan bool)

	consumer := New()

	if err = consumer.Connect(testHost, testPort); err != nil {
		t.Fatal("Failed to connect: ", err)
	}

	rec := make(chan []string)

	go func() {
		select {
		case ls = <-rec:
			t.Logf("Got: %v\n", ls)
			ok <- true
			return
		}
	}()

	if err = consumer.PSubscribe(rec, "channel"); err != nil {
		t.Fatal("Failed to subscribe: ", err)
	}

	<-ok

	consumer.PUnsubscribe("channel")

	consumer.Quit()
}

func TestPublishAndPSubscribe(t *testing.T) {
	var err error
	var received map[int]bool
	var tests int

	wg := new(sync.WaitGroup)

	mu := new(sync.Mutex)

	tests = 10

	// Subscriber
	consumer := New()

	if err = consumer.Connect(testHost, testPort); err != nil {
		t.Fatal("Failed to connect: ", err)
	}

	rec := make(chan []string, tests)

	psubscribed := make(chan bool)

	go func() {
		received = make(map[int]bool)
		for {
			// Receiving messages.
			select {
			case ls := <-rec:
				if ls[0] == `pmessage` {
					i, _ := strconv.Atoi(ls[3])
					received[i] = true
					// Marking job as done.
					wg.Done()
				} else if ls[0] == `psubscribe` {
					psubscribed <- true
				} else if ls[0] == `punsubscribe` {
					return
				}
			}
		}
	}()

	if err = consumer.PSubscribe(rec, "channe?"); err != nil {
		t.Fatal("Subscribe() ", err)
	}

	// Waiting for subscription to happen.
	<-psubscribed

	// Publisher
	publisher := New()

	if err = publisher.Connect(testHost, testPort); err != nil {
		t.Fatal("Failed to connect: ", err)
	}

	// Publishing
	var clients int64
	for i := 0; i < tests; i++ {
		mu.Lock()
		wg.Add(1)
		clients, err = publisher.Publish("channel", i)
		if err != nil {
			t.Fatal("Publish(): ", err)
		}
		if clients < 1 {
			t.Fatal("We should have at least one subscribed client.")
		}
		mu.Unlock()
	}

	// Waiting jobs to finish.
	wg.Wait()

	if _, err = publisher.Quit(); err != nil {
		t.Fatal("Quit(): ", err)
	}

	if err = consumer.PUnsubscribe("channel"); err != nil {
		t.Fatal("PUnsubscribe(): ", err)
	}

	if _, err = consumer.Quit(); err != nil {
		t.Fatal("Quit(): ", err)
	}

	// Verifying data map.
	for i := 0; i < tests; i++ {
		if received[i] == false {
			t.Fatal(`The "test" map should be populated with true values.`)
		}
	}
}

/*
func TestTransactions(t *testing.T) {
	var ls []interface{}
	var err error
	var s string
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	client.Del("mykey")

	client.Multi()

	client.Del("mykey")
	client.Set("mykey", 1)
	client.Incr("mykey")

	client.Discard()

	client.Multi()

	client.Set("mykey", 10)
	client.Incr("mykey")

	ls, err = client.Exec()

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed.")
	}

	s, _ = client.Get("mykey")

	if s != "11" {
		t.Fatalf("Failed.")
	}

}
*/

func TestEval(t *testing.T) {
	var ls []string
	var h string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	ls, err = client.Eval("return {KEYS[1],KEYS[2],ARGV[1],ARGV[2]}", 2, "key1", "key2", "first", "second")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) == 0 {
		t.Fatalf("Failed")
	}

	h, err = client.ScriptLoad("return {KEYS[1],KEYS[2],ARGV[1],ARGV[2]}")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	ls, err = client.ScriptExists("return {KEYS[1],KEYS[2],ARGV[1],ARGV[2]}")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) == 0 {
		t.Fatalf("Failed")
	}

	ls, err = client.EvalSHA(h, 2, "key1", "key2", "first", "second")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) == 0 {
		t.Fatalf("Failed")
	}

}

func TestRandom(t *testing.T) {
	var s string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Getting a random key
	s, err = client.RandomKey()

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(s) == 0 {
		t.Fatalf("Failed")
	}
}

func TestRename(t *testing.T) {
	var s string
	var b bool
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Setting key
	client.Set("mykey", "Hello")

	// Renaming
	s, err = client.Rename("mykey", "myotherkey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Getting value
	s, err = client.Get("myotherkey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "Hello" {
		t.Fatalf("Failed")
	}

	// Setting key
	client.Set("mykey", "taken")

	// Renaming if not exists, but it does.
	b, err = client.RenameNX("myotherkey", "mykey")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == true {
		t.Fatalf("Failed")
	}

}

func TestSort(t *testing.T) {
	var el []string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Deleting key
	client.Del("mylist")

	// Pushing stuff into key
	client.LPush("mylist", 5)
	client.LPush("mylist", 1)
	client.LPush("mylist", 4)
	client.LPush("mylist", -1)
	client.LPush("mylist", 9)

	// Sorting
	el, err = client.Sort("mylist")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(el) != 5 {
		t.Fatalf("Failed")
	}

	if el[0] != "-1" {
		t.Fatalf("Failed")
	}

}

func TestObject(t *testing.T) {
	var s string
	var i int64
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	client.Del("mylist")

	i, err = client.LPush("mylist", "Hello world")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	s, err = client.Object("refcount", "mylist")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "1" {
		t.Fatalf("Failed")
	}

	s, err = client.Object("encoding", "mylist")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "ziplist" {
		t.Fatalf("Failed")
	}
}

func TestKeys(t *testing.T) {
	var b bool
	var k []string
	var s string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Multiple set
	s, err = client.MSet(
		"one", 1,
		"two", 2,
		"three", 3,
		"four", 4,
	)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Getting "o" keys
	k, err = client.Keys("*o*")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(k) < 3 {
		t.Fatalf("Failed")
	}

	// Getting "t" keys
	k, err = client.Keys("t??")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(k) < 1 {
		t.Fatalf("Failed")
	}

	// Getting all keys
	k, err = client.MGet("one", "two", "three", "four")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(k) != 4 {
		t.Fatalf("Failed")
	}

	// Deleting keys
	client.Del("one", "two", "three", "four")

	// Multiple set (NX)
	b, err = client.MSetNX(
		"one", 1,
		"two", 2,
		"three", 3,
		"four", 4,
	)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == false {
		t.Fatalf("Failed")
	}

	// Multiple set (NX)
	b, err = client.MSetNX(
		"one", 1,
		"two", 2,
		"three", 3,
		"four", 4,
	)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == true {
		t.Fatalf("Failed")
	}
}

func TestRange(t *testing.T) {
	var i int64
	var s string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Setting key
	client.Set("key1", "Hello World")

	// Overwriting range
	i, err = client.SetRange("key1", 6, "Redis")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 11 {
		t.Fatalf("Failed")
	}

	// Verifying
	s, _ = client.Get("key1")

	if s != "Hello Redis" {
		t.Fatalf("Failed")
	}

	i, err = client.Strlen("key1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 11 {
		t.Fatalf("Failed")
	}
}

func TextHashes(t *testing.T) {
	var i int64
	var b bool
	var s string
	var ls []string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Deleting hash
	client.Del("myhash")

	// Setting hash value
	b, err = client.HSet("myhash", "field1", "foo")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == false {
		t.Fatalf("Failed")
	}

	// Getting hash value
	s, err = client.HGet("myhash", "field1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "foo" {
		t.Fatalf("Failed")
	}

	// Getting all hash values
	ls, err = client.HGetAll("myhash")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 1 {
		t.Fatalf("Failed")
	}

	// Deleting hash value
	i, err = client.HDel("myhash", "field1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Deleting non-existent hash value
	i, err = client.HDel("myhash", "field2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 0 {
		t.Fatalf("Failed")
	}

	// Incrementing key value
	i, err = client.HIncrBy("myhash", "field", 10)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 10 {
		t.Fatalf("Failed")
	}

	// Incrementing key value (float)
	s, err = client.HIncrByFloat("myhash", "field", 1.01)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "11.01" {
		t.Fatalf("Failed")
	}

	// Getting all hash keys
	ls, err = client.HKeys("myhash")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	i, err = client.HLen("myhash")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != int(i) {
		t.Fatalf("Failed")
	}

	// Using multi get
	ls, err = client.HMGet("myhash", "field1", "field2", "nofield")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 3 {
		t.Fatalf("Failed")
	}

	// Using multi set
	s, err = client.HMSet("myhash", "field1", 1, "field2", 2)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Non-existent set
	b, err = client.HSetNX("myhash", "field1", 1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == true {
		t.Fatalf("Failed")
	}

	// Getting values
	ls, err = client.HVals("myhash")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 2 {
		t.Fatalf("Failed")
	}
}

func TestLists(t *testing.T) {
	var ls []string
	var s string
	var i int64
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	// Deleting lists
	client.Del("list1", "list2")

	client.RPush("list1", "a", "b", "c")
	client.RPush("list2", "a", "b", "c")

	// Blocking LPOP
	ls, err = client.BLPop(0, "list1", "list2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) == 0 {
		t.Fatalf("Failed")
	}

	// Blocking RPOP
	ls, err = client.BRPop(0, "list1", "list2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) == 0 {
		t.Fatalf("Failed")
	}

	// Deleting lists
	client.Del("list1", "list2")

	// Pushing
	client.RPush("list1", "a", "b", "c")

	// RPop and LPush
	s, err = client.RPopLPush("list1", "list2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "c" {
		t.Fatalf("Failed")
	}

	// Checking second key
	ls, err = client.LRange("list2", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 1 {
		t.Fatalf("Failed")
	}

	// Deleting lists
	client.Del("list1", "list2")

	// Pushing
	client.RPush("list1", "a", "b", "c")

	// RPop and LPush
	s, err = client.BRPopLPush("list1", "list2", 10)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "c" {
		t.Fatalf("Failed")
	}

	// Checking second key
	ls, err = client.LRange("list2", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(ls) != 1 {
		t.Fatalf("Failed")
	}

	// Deleting lists
	client.Del("list1", "list2")

	// Pushing
	client.LPush("list1", "a", "b", "c")

	// Getting list element by index
	s, err = client.LIndex("list1", 1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "b" {
		t.Fatalf("Failed")
	}

	// Deleting lists
	client.Del("list1", "list2")

	// Pushing
	client.RPush("list1", "Hello", "World")

	// Inserting
	i, err = client.LInsert("list1", "BEFORE", "World", "There")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 3 {
		t.Fatalf("Failed")
	}

	// List pop
	s, err = client.LPop("list1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "Hello" {
		t.Fatalf("Failed")
	}

	// List Length
	i, err = client.LLen("list1")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 2 {
		t.Fatalf("Failed")
	}

	// List pop
	i, err = client.LPushX("list1", "Hello")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 3 {
		t.Fatalf("Failed")
	}

	// Removing equal
	i, err = client.LRem("list1", 0, "World")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	// Setting by index
	s, err = client.LSet("list1", 1, "World")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Deleting lists
	client.Del("mylist")

	// Pushing
	client.RPush("mylist", "one", "two", "three")

	// Trimming
	s, err = client.LTrim("mylist", 1, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	// Popping
	s, err = client.RPop("mylist")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "three" {
		t.Fatalf("Failed")
	}

	// RPop and LPush
	s, err = client.RPopLPush("mylist", "list2")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "two" {
		t.Fatalf("Failed")
	}

}

func TestSetBit(t *testing.T) {
	var err error
	var i int64
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	client.Del("mykey")

	_, err = client.SetBit("mykey", 7, 1)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	i, err = client.GetBit("mykey", 0)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 0 {
		t.Fatalf("Failed")
	}

	i, err = client.GetBit("mykey", 7)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}
}

func TestBitOp(t *testing.T) {
	var err error
	var s string
	var i int64
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	client.Set("key1", "foobar")
	client.Set("key2", "foobax")

	i, err = client.BitOp("AND", "dest", "key1", "key2")

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if i != 6 {
		t.Fatalf("Failed.")
	}

	s, err = client.Get("dest")

	if s != "foobap" {
		t.Fatalf("Failed.")
	}

}

func TestRawList(t *testing.T) {
	var r int
	var items []int
	var sitems []string
	var err error
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	err = client.Command(nil, "DEL", "list")

	if err != nil {
		if err != ErrNilReply {
			t.Fatalf("Command failed: %s", err.Error())
		}
	}

	for i := 0; i < 10; i++ {
		err = client.Command(&r, "LPUSH", "list", i)
		if err != nil {
			t.Fatalf("Command failed: %s", err.Error())
		}
		if r != i+1 {
			t.Fatalf("Failed, expecting %d, got %d", i+1, r)
		}
	}

	err = client.Command(&items, "LRANGE", "list", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(items) != 10 {
		t.Fatalf("Failed")
	}

	err = client.Command(&sitems, "LRANGE", "list", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if len(sitems) != 10 {
		t.Fatalf("Failed")
	}

}

// https://github.com/gosexy/redis/issues/27
func Test_Issue27(t *testing.T) {
	var client *Client
	var err error

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	info, err := client.Info("all")

	if err != nil {
		t.Fatalf("Command failed: %q", err)
	}

	if len(info) == 0 {
		t.Fatalf("Failed to actually get data from INFO.")
	}
}

// See https://github.com/gosexy/redis/issues/38
func Test_Issue38(t *testing.T) {
	var client *Client
	var err error

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	client.Del("mylist")
	client.Set("mylist", 1)

	defer client.Close()

	if _, err = client.RPush("mylist", 1); err == nil {
		t.Fatalf("Expecting an error.")
	}

	if !strings.Contains(err.Error(), "WRONGTYPE") {
		t.Fatalf("Expecting WRONGTYPE error.")
	}
}

func TestQuit(t *testing.T) {
	var err error
	var s string
	var client *Client

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		t.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	s, err = client.Quit()

	if err != nil {
		t.Fatalf("Failed")
	}

	if s != "OK" {
		t.Fatal()
	}

	s, err = client.Quit()

	if err == nil {
		t.Fatalf("Did not fail.")
	}

	if s == "OK" {
		t.Fatal()
	}

	_, err = client.Set("foo", 1)

	if err == nil {
		t.Fatalf("Did not fail.")
	}
}

func BenchmarkConnect(b *testing.B) {
	var client *Client
	var err error

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	err = client.ConnectWithTimeout(testHost, testPort, time.Second*1)
	//err := client.Connect(testHost, testPort)

	if err != nil {
		b.Fatalf(err.Error())
	}
}

func BenchmarkPing(b *testing.B) {
	var client *Client
	var err error
	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()

	for i := 0; i < b.N; i++ {
		_, err = client.Ping()
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkSet(b *testing.B) {
	var client *Client
	var err error

	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()
	client.Del("hello")
	for i := 0; i < b.N; i++ {
		_, err = client.Set("hello", 1)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGet(b *testing.B) {
	var client *Client
	var err error
	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()
	for i := 0; i < b.N; i++ {
		_, err = client.Get("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkIncr(b *testing.B) {
	var client *Client
	var err error
	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()
	for i := 0; i < b.N; i++ {
		_, err = client.Incr("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkLPush(b *testing.B) {
	var client *Client
	var err error
	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()
	client.Del("hello")
	for i := 0; i < b.N; i++ {
		_, err = client.LPush("hello", i)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkLRange10(b *testing.B) {
	var client *Client
	var err error
	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()
	for i := 0; i < b.N; i++ {
		_, err = client.LRange("hello", 0, 10)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkLRange100(b *testing.B) {
	var client *Client
	var err error
	client = New()

	if err = client.Connect(testHost, testPort); err != nil {
		b.Fatalf("Client failed to connect to an up-and-running redis server: %q", err)
	}

	defer client.Close()
	for i := 0; i < b.N; i++ {
		_, err = client.LRange("hello", 0, 100)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}
