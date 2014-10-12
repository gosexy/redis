package main

import (
	"github.com/xiam/resp"
	"log"
	"menteslibres.net/gosexy/redis"
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

	for _, v := range res {
		switch m := v.(type) {
		case []interface{}:
			log.Printf("Got an array of type %s with %d elements (%v).\n", reflect.TypeOf(m).Kind(), len(m), string(m[0].(*resp.Message).Interface().([]byte)))
		case *resp.Message:
			log.Printf("Got value of kind %s (%v), we use the integer part: %d\n", reflect.TypeOf(m).Kind(), m, m.Integer)
		}
	}

	client.Quit()

}
