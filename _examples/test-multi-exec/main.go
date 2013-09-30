package main

import (
	"fmt"
	"log"
	"menteslibres.net/gosexy/redis"
	"menteslibres.net/gosexy/to"
	"reflect"
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

	for _, el := range res {
		switch v := el.(type) {
		case []interface{}:
			log.Printf("Got []interface{}")
			for i, vv := range v {
				log.Printf("Value at index %d (%v) has kind %s, we convert it to string: %v", i, vv, reflect.TypeOf(vv).Kind(), to.String(vv))
			}
		case interface{}:
			fmt.Printf("Got value of kind %s (%v), we convert it to string: %s\n", reflect.TypeOf(v).Kind(), v, to.String(v))
		}
	}

	client.Quit()

}
