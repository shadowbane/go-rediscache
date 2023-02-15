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
//  Returns:
//   - *redis.Client
func connect(host string, password string, database int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       database,
	})

	return client
}

// Init Redis Cache
//
// This will be the main entry point for Redis Cache
// It will connect to Redis Server and return RedisCache instance,
// which can be used to store and retrieve data from Redis Server
//
// Parameters:
//  - c: RedisConfig
// Returns:
//  - RedisCache
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

// Set store value to cache
// You can set expiration time in seconds, which will automatically
// delete the key after the time has passed
//
// Parameters:
//  - key: string
//  - value: interface{}
//  - expiration: int (second)
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

// Get stored value from cache
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

// Forget stored value from cache
func (rc *RedisCache) Forget(key string) error {
	operation := rc.Connection.Del(context.Background(), getKeyWithPrefix(rc.Config, key))

	if operation.Err() != nil {
		return operation.Err()
	}

	return nil
}

// Flush the cache in current database
// Warning: This will flush all records in current redis database
func (rc *RedisCache) Flush() error {
	operation := rc.Connection.FlushDB(context.Background())

	if operation.Err() != nil {
		return operation.Err()
	}

	return nil
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
