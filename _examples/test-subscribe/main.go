package main

import (
	"log"
	"menteslibres.net/gosexy/redis"
	"strings"
)

var host = "127.0.0.1"
var port = uint(6379)

var publisher *redis.Client

var consumer *redis.Client

func main() {
	var err error

	publisher = redis.New()

	err = publisher.Connect(host, port)

	if err != nil {
		log.Fatalf("Publisher failed to connect: %s\n", err.Error())
		return
	}

	log.Println("Publisher connected to redis-server.")

	consumer = redis.New()

	err = consumer.ConnectNonBlock(host, port)

	if err != nil {
		log.Fatalf("Consumer failed to connect: %s\n", err.Error())
		return
	}

	log.Println("Consumer connected to redis-server.")

	rec := make(chan []string)

	log.Printf("Consumer is now inside a go-routine.\n")
	go consumer.Subscribe(rec, "channel")

	log.Printf("Publisher will send messages in another go-routine\n")

	go func() {
		publisher.Publish("channel", "Hello world!")
		publisher.Publish("channel", "Do you know how to count?")

		for i := 0; i < 3; i++ {
			publisher.Publish("channel", i)
		}
	}()

	log.Printf("Reading subscription...\n")
	var ls []string

	for j := 0; j < 6; j++ {
		ls = <-rec
		log.Printf("Consumer received: %v\n", strings.Join(ls, ", "))
	}

	log.Printf("Done, quitting.\n")

	consumer.Quit()
	publisher.Quit()

}
