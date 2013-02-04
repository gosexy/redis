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
	client = New()
	err := client.ConnectWithTimeout(host, port, time.Second*1)
	if err != nil {
		t.Fatalf("Connect failed: %s", err.Error())
	}
	r, err := client.Ping()
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}
	fmt.Printf("PING: %v\n", r)
}

func TestSet(t *testing.T) {
	r, err := client.Set("foo", "hello world.")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}
	fmt.Printf("SET: %v\n", r)
}

func TestGet(t *testing.T) {
	var r string
	var err error

	r, err = client.Set("foo", "hello")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	r, err = client.Get("foo")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	if r != "hello" {
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
	var r int64
	var err error

	r, err = client.Del("counter")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	fmt.Printf("DEL: %v\n", r)

	r, err = client.Del("counter")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	fmt.Printf("DEL (again): %v\n", r)
}

func TestIncr(t *testing.T) {
	var r int64
	var err error

	r, err = client.Incr("counter")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}
	fmt.Printf("INCR: %v\n", r)

	r, err = client.Incr("counter")
	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}
	fmt.Printf("INCR (again): %v\n", r)
}

func TestList(t *testing.T) {
	var r int64
	var items []string
	var err error

	_, err = client.Del("list")

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	for i := 0; i < 10; i++ {
		r, err = client.LPush("list", fmt.Sprintf("element-%d", i))
		fmt.Printf("LPUSH: %v\n", r)
	}

	items, err = client.LRange("list", 0, -1)

	if err != nil {
		t.Fatalf("Command failed: %s", err.Error())
	}

	for _, item := range items {
		fmt.Printf("LRANGE -> %s\n", item)
	}

}

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
