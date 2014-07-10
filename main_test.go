package redis

import (
	"flag"
	"fmt"
	"log"
	"testing"
	"time"
)

var client *Client

var (
	testHost string
	testPort uint
)

func init() {
	// Getting host and port from command line.

	host := flag.String("host", "127.0.0.1", "Test hostname or address.")
	port := flag.Uint("port", 6379, "Port.")

	flag.Parse()

	testHost = *host
	testPort = *port

	log.Printf("Running tests against host %s:%d.\n", testHost, testPort)
}

func TestConnect(t *testing.T) {
	var err error

	client = New()

	// Attempting a valid connection.
	err = client.Connect(testHost, testPort)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}

	// Attempting to connect to port 0, probably closed...
	err = client.Connect(testHost, 0)
	if err == nil {
		t.Fatalf("Expecting a connection error.")
	}
}

func TestPing(t *testing.T) {
	var s string
	var err error

	client = New()

	err = client.ConnectWithTimeout(testHost, testPort, time.Second*1)

	if err != nil {
		t.Fatalf(err.Error())
	}

	s, err = client.Ping()

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if s != "PONG" {
		t.Fatalf("Failed")
	}

	client.Quit()
}

func TestPingAsync(t *testing.T) {
	var s string
	var err error

	client = New()

	err = client.ConnectNonBlock(testHost, testPort)

	if err != nil {
		t.Fatalf(err.Error())
	}

	s, err = client.Ping()

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if s != "PONG" {
		t.Fatalf("Failed")
	}

	client.Quit()

}

func TestSimpleSet(t *testing.T) {
	var s string
	var b bool
	var err error

	// We'll be reusing this client.
	err = client.Connect(testHost, testPort)

	if err != nil {
		t.Fatalf(err.Error())
	}

	s, err = client.Set("foo", "hello world.")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "OK" {
		t.Fatalf("Failed")
	}

	b, err = client.SetNX("foo", "exists!")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if b == true {
		t.Fatalf("Failed")
	}
}

func TestGet(t *testing.T) {
	var s string
	var err error

	client.Set("foo", "hello")

	s, err = client.Get("foo")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if s != "hello" {
		t.Errorf("Could not SET/GET value.")
	}

	// Deleting key.
	client.Del("foo")

	// Attempting to retrieve deleted key.
	s, err = client.Get("foo")

	if s != "" {
		t.Fatalf("Expecting an empty string.")
	}
	if err != ErrNilReply {
		t.Fatalf("Expecting a redis.ErrNilReply error.")
	}

	// Making sure https://github.com/gosexy/redis/issues/23 does not interfere
	// with https://github.com/gosexy/redis/issues/12.
	client.Del("test")

	client.HMSet("test", "123", "*", "456", "*", "789", "*")

	var vals []string
	vals, err = client.HMGet("test", "1232", "456")

	if err != nil {
		t.Fatalf("Expecting no error.")
	}

	if vals[0] != "" || vals[1] != "*" {
		t.Fatalf("Unexpected values.")
	}

}

func TestSetGetUnicode(t *testing.T) {
	var r string
	var err error

	r, err = client.Set("heart", "♥")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	r, err = client.Get("heart")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if r != "♥" {
		t.Errorf("Could not SET/GET binary value.")
	}
}

func TestDel(t *testing.T) {
	var i int64
	var err error

	client.Set("counter", 0)

	i, err = client.Del("counter")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 1 {
		t.Fatalf("Failed")
	}

	i, err = client.Del("counter")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if i != 0 {
		t.Fatalf("Failed")
	}
}

func TestList(t *testing.T) {
	var items []string
	var err error

	_, err = client.Del("list")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
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

func TestPublish(t *testing.T) {
	var err error

	publisher := New()

	err = publisher.Connect(testHost, testPort)

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Publishing
	for i := 0; i < 200; i++ {
		_, err = publisher.Publish("channel", i)
		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}
	}

	publisher.Quit()
}

func TestSubscriptions(t *testing.T) {
	var ls []string

	var err error

	consumer := New()

	err = consumer.ConnectNonBlock(testHost, testPort)

	consumer.Set("test", "TestSubscriptions")

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	rec := make(chan []string)

	go func() {
		select {
		case ls = <-rec:
			t.Logf("Got: %v\n", ls)
		}
	}()

	go consumer.Subscribe(rec, "channel")

	consumer.Unsubscribe("channel")

	consumer.Quit()
}

func TestPSubscriptions(t *testing.T) {
	var ls []string

	var err error

	consumer := New()

	err = consumer.ConnectNonBlock(testHost, testPort)

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	consumer.Set("test", "TestPSubscriptions")

	rec := make(chan []string)

	go func() {
		select {
		case ls = <-rec:
			t.Logf("Got: %v (%d)\n", ls)
		}
	}()

	go consumer.PSubscribe(rec, "channel")

	consumer.PUnsubscribe("channel")

	consumer.Quit()
}

func TestTransactions(t *testing.T) {
	var ls []interface{}
	var err error
	var s string

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

func TestEval(t *testing.T) {
	var ls []string
	var h string
	var err error

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

	err = client.Command(nil, "DEL", "list")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	for i := 0; i < 10; i++ {
		err = client.Command(&r, "LPUSH", "list", i)
		if err != nil {
			t.Fatalf("Command failed: %s", err.Error())
		}
		if r == 0 {
			t.Fatalf("Failed")
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
func TestInfo(t *testing.T) {
	info, err := client.Info("all")

	if err != nil {
		t.Fatalf("Command failed: %q", err)
	}

	if len(info) == 0 {
		t.Fatalf("Failed to actually get data from INFO.")
	}
}

func TestQuit(t *testing.T) {
	var err error

	_, err = client.Quit()

	if err != nil {
		t.Fatalf("Failed")
	}

	_, err = client.Quit()

	if err == nil {
		t.Fatalf("Did not fail.")
	}

	_, err = client.Set("foo", 1)

	if err == nil {
		t.Fatalf("Did not fail.")
	}
}

func BenchmarkConnect(b *testing.B) {
	client = New()

	err := client.ConnectWithTimeout(testHost, testPort, time.Second*1)
	//err := client.ConnectNonBlock(testHost, testPort)

	if err != nil {
		b.Fatalf(err.Error())
	}
}

func BenchmarkPing(b *testing.B) {
	var err error
	client.Del("hello")
	for i := 0; i < b.N; i++ {
		_, err = client.Ping()
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = client.Set("hello", 1)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = client.Get("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkIncr(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = client.Incr("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkLPush(b *testing.B) {
	var err error
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
	var err error
	for i := 0; i < b.N; i++ {
		_, err = client.LRange("hello", 0, 10)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkLRange100(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = client.LRange("hello", 0, 100)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}
