package rediscache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	_ "github.com/shadowbane/go-logger"
)

type RedisCache struct {
	Connection *redis.Client
}

func connect(host string, password string, database int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       database,
	})

	return client
}

func Init(c *RedisConfig) *RedisCache {
	redisClient := connect(c.GetConnection(), c.Password, c.DB)

	ping := redisClient.Ping(context.TODO())
	if ping.Err() != nil {
		zap.S().Errorf("Error connecting to Redis Server: %s", ping.Err())
	}

	zap.S().Debug("Connected to Redis Server")

	return &RedisCache{
		Connection: redisClient,
	}
}
