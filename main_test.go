package redis

import (
	"fmt"
	"testing"
	"time"
)

var host = "127.0.0.1"
var port = uint(6379)

func TestConnect(t *testing.T) {
	client := New()
	err := client.ConnectWithTimeout(host, port, time.Second*1)
	if err != nil {
		t.Errorf("Connect failed: %s", err.Error())
	}
	r, err := client.Ping()
	if err != nil {
		t.Errorf("Command failed: %s", err.Error())
	}
	fmt.Printf("PING %v\n", r)
}
