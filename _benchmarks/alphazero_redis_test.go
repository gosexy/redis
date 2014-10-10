package main

import (
	redis "github.com/alphazero/Go-Redis"
	"menteslibres.net/gosexy/to"
	"testing"
)

var alphazeroRedisClient redis.Client

func TestAlphazeroRedisConnect(t *testing.T) {
	var err error

	spec := redis.DefaultSpec()

	spec.Host(host)

	alphazeroRedisClient, err = redis.NewSynchClientWithSpec(spec)

	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	err = alphazeroRedisClient.Ping()

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

}

func BenchmarkAlphazeroRedisPing(b *testing.B) {
	var err error
	alphazeroRedisClient.Del("hello")
	for i := 0; i < b.N; i++ {
		err = alphazeroRedisClient.Ping()
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkAlphazeroRedisSet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		err = alphazeroRedisClient.Set("hello", to.Bytes(1))
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkAlphazeroRedisGet(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = alphazeroRedisClient.Get("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkAlphazeroRedisIncr(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = alphazeroRedisClient.Incr("hello")
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkAlphazeroRedisLPush(b *testing.B) {
	var err error
	alphazeroRedisClient.Del("hello")

	for i := 0; i < b.N; i++ {
		err = alphazeroRedisClient.Lpush("hello", to.Bytes(i))
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkAlphazeroRedisLRange10(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = alphazeroRedisClient.Lrange("hello", 0, 10)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}

func BenchmarkAlphazeroRedisLRange100(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		_, err = alphazeroRedisClient.Lrange("hello", 0, 100)
		if err != nil {
			b.Fatalf(err.Error())
			break
		}
	}
}
