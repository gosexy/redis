package main

import (
	"fmt"
	"log"
	"menteslibres.net/gosexy/redis"
)

var host = "127.0.0.1"
var port = uint(6379)

var client *redis.Client

func main() {
	var err error

	client = redis.New()

	err = client.Connect(host, port)
	if err != nil {
		log.Printf(err.Error())
	}

	compact := "HELLO"
	max := len(compact)

	client.Multi()
	client.Del("keywordlookup")
	client.Del("keywordlookup2")

	for i := 1; i <= max; i++ {
		s := compact[:i]
		client.ZAdd("keywordlookup", 0, s)
		client.ZAdd("keywordlookup2", 0, s)
	}

	client.ZInterStore("keywordmatch", 2, "keywordlookup", "keywordlookup2")
	client.ZRevRange("keywordmatch", 0, 0)
	res, err := client.Exec()

	if err != nil {
		log.Printf(err.Error())
	}

	fmt.Printf("%v\n", res)

	client.Quit()

}
