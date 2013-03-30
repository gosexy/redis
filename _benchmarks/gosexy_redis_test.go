package main

import (
	"menteslibres.net/gosexy/redis"
	"testing"
	"time"
)

var gosexyRedisClient *redis.Client

func TestGosexyRedisConnect(t *testing.T) {
	var s string
	var err error

	gosexyRedisClient = redis.New()

	err = gosexyRedisClient.ConnectWithTimeout(host, port, time.Second*1)

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	s, err = gosexyRedisClient.Ping()

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if s != "PONG" {
		t.Fatalf("Failed")
	}
}

func BenchmarkGosexyRedisPing(b *testing.B) {
	var err error
	gosexyRedisClient.Del("hello")
	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.Ping()
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGosexyRedisSet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.Set("hello", 1)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGosexyRedisGet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.Get("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGosexyRedisIncr(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.Incr("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGosexyRedisLPush(b *testing.B) {
	var err error
	gosexyRedisClient.Del("hello")

	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.LPush("hello", i)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGosexyRedisLRange10(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.LRange("hello", 0, 10)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkGosexyRedisLRange100(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = gosexyRedisClient.LRange("hello", 0, 100)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}
