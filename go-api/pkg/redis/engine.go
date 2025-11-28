package redis

import (
	"github.com/redis/go-redis/v9"
	"julianmorley.ca/con-plar/prog2270/pkg/global"
)

func RedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     global.GetEnvOrDefault("REDIS_ADDRESS", "localhost:6379"),
		Password: global.GetEnvOrDefault("REDIS_PASSWORD", ""),
		DB:       0,
		Protocol: 2,
	})
}
