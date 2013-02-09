package redis

import (
	"fmt"
	"testing"
	"time"
)

var host = "127.0.0.1"
var port = uint(6379)
var client *Client

func TestConnect(t *testing.T) {
	var s string
	var err error

	client = New()

	err = client.ConnectWithTimeout(host, port, time.Second*1)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	s, err = client.Ping()

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if s != "PONG" {
		t.Fatalf("Failed")
	}
}

func TestSet(t *testing.T) {
	var s string
	var b bool
	var err error

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

	if i < 1000 {
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

/*
func TestRawList(t *testing.T) {
	var r int
	var items []int
	var sitems []string
	var err error

	fmt.Printf("Raw commands\n")

	err = client.Command(nil, "DEL", "list")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	for i := 0; i < 10; i++ {
		err = client.Command(&r, "LPUSH", "list", i)
		if err != nil {
			t.Fatalf("Command failed: %s", err.Error())
		}
		fmt.Printf("LPUSH: %v\n", r)
	}

	err = client.Command(&items, "LRANGE", "list", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	for _, item := range items {
		fmt.Printf("LRANGE -> %d\n", item)
	}

	err = client.Command(&sitems, "LRANGE", "list", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	for _, sitem := range sitems {
		fmt.Printf("LRANGE -> %s\n", sitem)
	}

}
*/
