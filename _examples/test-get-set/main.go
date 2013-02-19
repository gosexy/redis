package main

import (
	"github.com/gosexy/redis"
	"log"
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
		log.Fatalf("Connect failed: %s\n", err.Error())
		return
	}

	log.Println("Connected to redis-server.")

	// DEL hello
	log.Printf("DEL hello\n")
	client.Del("hello")

	// SET hello 1
	log.Printf("SET hello 1\n")
	client.Set("hello", 1)

	// INCR hello
	log.Printf("INCR hello\n")
	client.Incr("hello")

	// GET hello
	log.Printf("GET hello\n")
	s, err = client.Get("hello")

	if err != nil {
		log.Fatalf("Could not GET: %s\n", err.Error())
		return
	}

	log.Printf("> hello = %s\n", s)

	client.Quit()
}
