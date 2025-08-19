package cache

import (
	"github.com/redis/go-redis/v9"
)

// Интерфейс для кэша
type Cache interface {
}

// Реализация через Redis
type RedisCache struct {
	client *redis.Client
}

type RedisConfig struct {
	Host string
	DB   int
}

func NewRedisCache(redisConfig RedisConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Host,
		Password: "",
		DB:       redisConfig.DB,
	})
	return &RedisCache{client: client}
}
