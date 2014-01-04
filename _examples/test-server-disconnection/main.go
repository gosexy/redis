package main

import (
	"fmt"
	"log"
	"menteslibres.net/gosexy/redis"
)

var host = "127.0.0.1"
var port = uint(6379)

var consumer *redis.Client

func main() {
	var err error

	consumer = redis.New()

	err = consumer.ConnectNonBlock(host, port)

	if err != nil {
		log.Fatalf("Consumer failed to connect: %s\n", err.Error())
		return
	}

	log.Println("Consumer connected to redis-server.")

	// Subscribing to a channel.
	log.Println("Client is subscribed to a test channel. Try stopping redis-server...")
	rec := make(chan []string, 10)
	err = consumer.Subscribe(rec, "channel")

	if err != nil {
		fmt.Printf("Unsubscribed with error: %s\n", err.Error())
	}

	consumer.Quit()
}
