package rediscache

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"log"
	"time"

	_ "github.com/shadowbane/go-logger"
)

type RedisCache struct {
	Connection *redis.Client
	Config     *RedisConfig
}

// Connect to Redis Server
//
//  Parameters:
//   - host: Redis Server Host
//   - password: Redis Server Password
//   - database: Redis Server Database int
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
		log.Fatalf("Error connecting to Redis Server: %s", ping.Err())
	}

	return &RedisCache{
		Connection: redisClient,
		Config:     c,
	}
}

func (rc *RedisCache) Set(key string, value interface{}, expiration int) error {

	exp := time.Duration(expiration) * time.Second

	valueToStore, err := ToJson(value)
	if err != nil {
		return err
	}

	set := rc.Connection.Set(context.Background(), getKeyWithPrefix(rc.Config, key), valueToStore, exp)
	if set.Err() != nil {
		return set.Err()
	}

	return nil
}

func (rc *RedisCache) Get(key string) (interface{}, error) {
	operation := rc.Connection.Get(context.Background(), getKeyWithPrefix(rc.Config, key))

	if operation.Err() != nil {
		return nil, operation.Err()
	}

	result, err := operation.Result()
	if err != nil {
		return nil, err
	}

	if IsJson(result) {
		iface, _ := ToInterface(result)
		return iface, nil
	}

	return result, nil
}

// Has checks if the key exists in the cache
func (rc *RedisCache) Has(key string) bool {
	operation := rc.Connection.Exists(context.Background(), getKeyWithPrefix(rc.Config, key))

	if operation.Err() != nil {
		return false
	}

	result, err := operation.Result()
	if err != nil {
		return false
	}

	return result > 0
}

func IsJson(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func ToJson(value interface{}) (string, error) {
	jsonString, err := json.Marshal(value)

	if err != nil {
		return "", err
	}

	return string(jsonString), nil
}

func ToInterface(value string) (interface{}, error) {
	var result interface{}
	err := json.Unmarshal([]byte(value), &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func getKeyWithPrefix(c *RedisConfig, value string) string {
	key := c.Prefix + ":" + value

	return key
}
