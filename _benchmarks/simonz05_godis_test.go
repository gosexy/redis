package main

import (
	"fmt"
	"github.com/simonz05/godis/redis"
	"testing"
)

var simonz05GodisClient *redis.Client

func TestSimonz05GodisConnect(t *testing.T) {
	simonz05GodisClient = redis.New(fmt.Sprintf("tcp:%s:%d", host, port), 0, "")

	b, err := simonz05GodisClient.Ping()

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if string(b) != "PONG" {
		t.Fatalf("Failed")
	}
}

func BenchmarkSimonz05GodisPing(b *testing.B) {
	var err error
	simonz05GodisClient.Del("hello")
	for i := 0; i < b.N; i++ {
		_, err = simonz05GodisClient.Ping()
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSimonz05GodisSet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		err = simonz05GodisClient.Set("hello", 1)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSimonz05GodisGet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = simonz05GodisClient.Get("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSimonz05GodisIncr(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = simonz05GodisClient.Incr("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSimonz05GodisLPush(b *testing.B) {
	var err error
	simonz05GodisClient.Del("hello")
	for i := 0; i < b.N; i++ {
		_, err = simonz05GodisClient.Lpush("hello", i)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSimonz05GodisLRange10(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = simonz05GodisClient.Lrange("hello", 0, 10)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkSimonz05GodisLRange100(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = simonz05GodisClient.Lrange("hello", 0, 100)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}
