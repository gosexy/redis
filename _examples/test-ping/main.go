package main

import (
	"fmt"
	"github.com/gosexy/redis"
)

var host = "127.0.0.1"
var port = uint(6379)

var client *redis.Client

func main() {
	var s string
	var err error

	client = redis.New()

	err = client.Connect(host, port)
	if err != nil {
		fmt.Printf("Connect failed: %v", err)
		return
	}

	fmt.Printf("Connected!\n")

	fmt.Printf("PING ->\n")

	s, err = client.Ping()

	fmt.Printf("PONG <-\n")

	if err != nil {
		fmt.Printf("Command failed: %v", err)
		return
	}

	if s != "PONG" {
		fmt.Printf("Failed")
		return
	}
}
