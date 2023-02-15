package rediscache

import (
	"flag"
	"os"
	"strconv"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Prefix   string
}

func LoadEnvForRedis() *RedisConfig {
	rc := &RedisConfig{}

	Host := getenv("REDIS_HOST", "localhost")
	Port := getenv("REDIS_PORT", "6379")
	Password := getenv("REDIS_PASSWORD", "")
	Database, _ := strconv.Atoi(getenv("REDIS_DB", "0"))

	flag.StringVar(&rc.Host, "Redis Host", Host, "Redis Host")
	flag.StringVar(&rc.Port, "Redis Port", Port, "Redis Port")
	flag.StringVar(&rc.Password, "Redis Password", Password, "Redis Password")
	flag.IntVar(&rc.DB, "Redis DB", Database, "Redis DB")
	flag.StringVar(&rc.Prefix, "Redis Cache Prefix", getenv("REDIS_PREFIX", "app-data"), "Redis Cache Prefix")

	return rc
}

// getenv get environment variable or fallback to default value if not set
func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func (c *RedisConfig) GetConnection() string {
	// merge host and port
	return c.Host + ":" + c.Port
}
