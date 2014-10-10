package main

import (
	"fmt"
	"github.com/tideland/godm/v3/redis"
	"testing"
)

var tcglRedisClient *redis.Connection

func TestTcglRedisConnect(t *testing.T) {
	option := redis.TcpConnection(fmt.Sprintf(`%s:%d`, host, port), 0)
	db, err := redis.Open(option)

	if err != nil {
		t.Fatal(err)
	}

	tcglRedisClient, err = db.Connection()

	if err != nil {
		t.Fatal(err)
	}

	s, err := tcglRedisClient.DoString("PING")

	if err != nil {
		t.Fatalf("Do failed: %q", err)
	}

	if s != "+PONG" {
		t.Fatalf("Failed")
	}
}

func BenchmarkTcglRedisPing(b *testing.B) {
	var err error
	tcglRedisClient.Do("DEL", "hello")
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("PING")
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkTcglRedisSet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("SET", "hello", 1)
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkTcglRedisGet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("GET", "hello")
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkTcglRedisIncr(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("INCR", "hello")
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkTcglRedisLPush(b *testing.B) {
	var err error
	tcglRedisClient.Do("DEL", "hello")
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("LPUSH", "hello", i)
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkTcglRedisLRange10(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("LRANGE", "hello", 0, 10)
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}

func BenchmarkTcglRedisLRange100(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = tcglRedisClient.Do("LRANGE", "hello", 0, 100)
		if err != nil {
			b.Fatal(err)
			break
		}
	}
}
