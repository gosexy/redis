package main

import (
	"log"
	"menteslibres.net/gosexy/redis"
	"strings"
	"time"
)

var host = "127.0.0.1"
var port = uint(6379)

func spawnPublisher() error {
	var err error
	var publisher *redis.Client

	publisher = redis.New()

	err = publisher.Connect(host, port)

	if err != nil {
		log.Fatalf("Publisher failed to connect: %s\n", err.Error())
		return err
	}

	log.Println("Publisher connected to redis-server.")

	log.Println("Publishing some messages...")

	publisher.Publish("channel", "Hello world!")
	publisher.Publish("channel", "Do you know how to count?")

	for i := 0; i < 3; i++ {
		publisher.Publish("channel", i)
	}

	log.Printf("Closing publisher...\n")

	publisher.Quit()

	return nil
}

func spawnConsumer() error {
	var err error
	var consumer *redis.Client

	consumer = redis.New()

	err = consumer.ConnectNonBlock(host, port)

	if err != nil {
		log.Fatalf("Consumer failed to connect: %s\n", err.Error())
		return err
	}

	log.Println("Consumer connected to redis-server.")

	rec := make(chan []string)

	log.Printf("Consumer will read exactly 6 messages before quitting.\n")

	go consumer.Subscribe(rec, "channel")

	var ls []string

	for j := 0; j < 6; j++ {
		ls = <-rec
		log.Printf("Consumer received message[%d]: %v\n", j, strings.Join(ls, ", "))
	}

	log.Printf("Closing consumer...\n")

	consumer.Quit()

	return nil
}

func main() {

	log.Printf("Spawning consumer into a go-routine.\n")
	go spawnConsumer()

	log.Printf("Spawning publisher into a go-routine.\n")
	go spawnPublisher()

	log.Printf("Waiting 3 secs...\n")
	time.Sleep(time.Second * 3)

}
