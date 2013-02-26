package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"testing"
)

var garyburdRedigoClient redis.Conn

func TestGaryburdRedigoConnect(t *testing.T) {
	var s interface{}
	var err error

	garyburdRedigoClient, err = redis.Dial("tcp", fmt.Sprintf("%s:%d", host, port))

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	s, err = garyburdRedigoClient.Do("PING")

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if s.(string) != "PONG" {
		t.Fatalf("Failed")
	}
}

func BenchmarkGaryburdRedigoPing(b *testing.B) {
	var err error
	garyburdRedigoClient.Do("DEL", "hello")
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("PING")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGaryburdRedigoSet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("SET", "hello", 1)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGaryburdRedigoGet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("GET", "hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGaryburdRedigoIncr(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("INCR", "hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGaryburdRedigoLPush(b *testing.B) {
	var err error
	garyburdRedigoClient.Do("DEL", "hello")
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("LPUSH", "hello", i)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGaryburdRedigoLRange10(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("LRANGE", "hello", 0, 10)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGaryburdRedigoLRange100(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = garyburdRedigoClient.Do("LRANGE", "hello", 0, 100)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}
