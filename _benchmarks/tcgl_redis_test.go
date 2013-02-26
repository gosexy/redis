package main

import (
	"code.google.com/p/tcgl/redis"
	"fmt"
	"testing"
)

var tcglRedisClient *redis.Database

func TestTcglRedisConnect(t *testing.T) {
	tcglRedisClient = redis.Connect(redis.Configuration{Address: fmt.Sprintf("%s:%d", host, port)})

	res := tcglRedisClient.Command("PING")

	if res.Error() != nil {
		t.Fatalf("Command failed: %v", res.Error())
	}

	if res.ValueAsString() != "PONG" {
		t.Fatalf("Failed")
	}
}

func BenchmarkTcglRedisPing(b *testing.B) {
	var res *redis.ResultSet
	tcglRedisClient.Command("DEL", "hello")
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("PING")
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}

func BenchmarkTcglRedisSet(b *testing.B) {
	var res *redis.ResultSet
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("SET", "hello", 1)
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}

func BenchmarkTcglRedisGet(b *testing.B) {
	var res *redis.ResultSet
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("GET", "hello")
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}

func BenchmarkTcglRedisIncr(b *testing.B) {
	var res *redis.ResultSet
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("INCR", "hello")
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}

func BenchmarkTcglRedisLPush(b *testing.B) {
	var res *redis.ResultSet
	tcglRedisClient.Command("DEL", "hello")
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("LPUSH", "hello", i)
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}

func BenchmarkTcglRedisLRange10(b *testing.B) {
	var res *redis.ResultSet
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("LRANGE", "hello", 0, 10)
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}

func BenchmarkTcglRedisLRange100(b *testing.B) {
	var res *redis.ResultSet
	for i := 0; i < b.N; i++ {
		res = tcglRedisClient.Command("LRANGE", "hello", 0, 100)
		if res.Error() != nil {
			b.Fatalf(res.Error().Error())
			break
		}
	}
}
