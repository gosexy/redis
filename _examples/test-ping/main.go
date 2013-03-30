package main

import (
	"menteslibres.net/gosexy/redis"
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

	log.Printf("Sending PING...\n")
	s, err = client.Ping()

	if err != nil {
		log.Fatalf("Could not ping: %s\n", err.Error())
		return
	}

	log.Printf("Received %s!\n", s)

	client.Quit()

}
