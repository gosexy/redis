package main

import (
	"github.com/gosexy/redis"
	"log"
)

var host = "127.0.0.1"
var port = uint(6379)

var client *redis.Client

func main() {
	var err error

	client = redis.New()

	err = client.Connect(host, port)

	if err != nil {
		log.Fatalf("Connect failed: %s\n", err.Error())
		return
	}

	log.Println("Connected to redis-server.")

	log.Printf("DEL mylist")
	client.Del("mylist")

	for i := 0; i < 10; i++ {
		log.Printf("RPUSH mylist %d\n", i*2)
		client.RPush("mylist", i*2)
	}

	log.Printf("LRANGE mylist 0 5")
	var ls []string
	ls, err = client.LRange("mylist", 0, 5)

	for i := 0; i < len(ls); i++ {
		log.Printf("> mylist[%d] = %s\n", i, ls[i])
	}

	client.Quit()

}
