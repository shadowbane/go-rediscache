package rediscache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
	"time"
)

var redisServer *miniredis.Miniredis

func Setup(t *testing.T) {
	redisServer = miniredis.RunT(t)
}

func initRedis() *RedisCache {
	cfg := &RedisConfig{
		Host:     redisServer.Host(),
		Port:     redisServer.Port(),
		Password: "",
		DB:       0,
		Prefix:   "app-data",
	}

	return Init(cfg)
}

func teardown() {
	redisServer.Close()
}

func testOsExit(t *testing.T, funcName string, testFunc func(*testing.T)) {
	if os.Getenv(funcName) == "1" {
		testFunc(t)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run="+funcName)
	cmd.Env = append(os.Environ(), funcName+"=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatal("subprocess ran successfully, want non-zero exit status")
}

func TestInit(t *testing.T) {
	Setup(t)
	defer teardown()

	cfg := &RedisConfig{
		Host:     redisServer.Host(),
		Port:     redisServer.Port(),
		Password: "",
		DB:       0,
		Prefix:   "app-data",
	}

	t.Run("Can be initialized", func(t *testing.T) {
		assert.IsTypef(t, &RedisCache{}, Init(cfg), "Init() = %v, want %v", Init(cfg), &RedisCache{})
	})

	t.Run("Throws fatal error when connection to Redis Server failed", func(t *testing.T) {
		testOsExit(t, "TestInit", func(t *testing.T) {
			Init(&RedisConfig{
				Host:     "localhost",
				Port:     "6179",
				Password: "",
				DB:       0,
				Prefix:   "app-data",
			})
		})
	})
}

func TestIsJson(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Test IsJson", args{"{\"test\": \"test\"}"}, true},
		{"Test IsJson", args{"{\"test\": \"test\""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJson(tt.args.str); got != tt.want {
				t.Errorf("IsJson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadEnvForRedis(t *testing.T) {
	tests := []struct {
		name string
		want *RedisConfig
	}{
		{"Test Load Env", &RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
			Prefix:   "app-data",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LoadEnvForRedis(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadEnvForRedis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	Setup(t)
	defer teardown()
	rCache := initRedis()

	type args struct {
		key string
	}
	tests := []struct {
		name      string
		args      args
		want      interface{}
		wantErr   bool
		beforeRun func()
	}{
		{
			"Cache Get",
			args{key: "test"},
			"this is test value",
			false,
			func() {
				rCache.Connection.Set(
					context.Background(),
					getKeyWithPrefix(rCache.Config, "test"),
					"this is test value",
					time.Duration(5)*time.Second,
				)
			},
		},
		{
			"Cache Get",
			args{key: "test"},
			nil,
			true,
			func() {
				redisServer.FastForward(time.Duration(6) * time.Second)

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// before run
			tt.beforeRun()

			got, err := rCache.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisCache_Has(t *testing.T) {
	Setup(t)
	defer teardown()

	rCache := initRedis()
	rCache.Connection.Set(
		context.Background(),
		getKeyWithPrefix(rCache.Config, "test"),
		"test",
		time.Duration(5)*time.Second,
	)

	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Cache Has", args{key: "test"}, true},
		{"Cache Has", args{key: "testing"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rCache.Has(tt.args.key); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisCache_Set(t *testing.T) {
	Setup(t)
	defer teardown()

	rCache := initRedis()

	type args struct {
		key        string
		value      interface{}
		expiration int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Can set the cache",
			args{
				key:        "testing",
				value:      "This is just a test",
				expiration: 1,
			},
			true,
		},
		{
			"Cache is not exists",
			args{
				key:        "test-some-val",
				value:      "This is just a test",
				expiration: 1,
			},
			false,
		},
		{
			"Cache is exists and expired",
			args{
				key:        "test-3",
				value:      "This is just a test",
				expiration: 1,
			},
			false,
		},
	}

	t.Run(tests[0].name, func(t *testing.T) {
		_ = rCache.Set(tests[0].args.key, tests[0].args.value, tests[0].args.expiration)

		assertion, err := rCache.Connection.Exists(context.Background(), getKeyWithPrefix(rCache.Config, tests[0].args.key)).Result()
		t.Logf("assertion: %v", assertion)
		if err != nil {
			t.Errorf("Set() error = %v, want %v", err, tests[0].want)
		}

		assert.Equal(t, tests[0].want, assertion > 0, "Set() error. Cache '%s' with value '%v' is not exist", tests[0].args.key, tests[0].args.value)
		value, _ := rCache.Get(tests[0].args.key)
		assert.Equal(t, tests[0].args.value, value, "Set() error. Cache '%s' with value '%v' is not exist", tests[0].args.key, tests[0].args.value)
	})

	t.Run(tests[1].name, func(t *testing.T) {
		result := rCache.Has(tests[1].args.key)

		assert.Equal(t, tests[1].want, result, "Set() error. Cache '%s' with value '%v' is exist", tests[1].args.key, tests[1].args.value)
	})

	t.Run(tests[2].name, func(t *testing.T) {
		_ = rCache.Set(tests[2].args.key, tests[2].args.value, tests[2].args.expiration)
		assertion, err := rCache.Connection.Exists(context.Background(), getKeyWithPrefix(rCache.Config, tests[2].args.key)).Result()

		if err != nil {
			t.Errorf("Set() error = %v, want %v", err, tests[2].want)
		}

		assert.Equal(t, !tests[2].want, assertion > 0, "Set() error. Cache '%s' with value '%v' is not exist", tests[2].args.key, tests[2].args.value)

		// wait for cache expired
		redisServer.FastForward(2 * time.Second)

		assertion, err = rCache.Connection.Exists(context.Background(), getKeyWithPrefix(rCache.Config, tests[2].args.key)).Result()
		assert.Equal(t, tests[2].want, assertion > 0, "Set() error. Cache '%s' with value '%v' is exist", tests[2].args.key, tests[2].args.value)
	})
}

func TestRedisConfig_GetConnection(t *testing.T) {
	type fields struct {
		Host     string
		Port     string
		Password string
		DB       int
		Prefix   string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"Should give host and port", fields{Host: "localhost", Port: "6379"}, "localhost:6379"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RedisConfig{
				Host:     tt.fields.Host,
				Port:     tt.fields.Port,
				Password: tt.fields.Password,
				DB:       tt.fields.DB,
				Prefix:   tt.fields.Prefix,
			}
			if got := c.GetConnection(); got != tt.want {
				t.Errorf("GetConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToInterface(t *testing.T) {
	type args struct {
		value string
	}

	// convert 1 to float
	flt, _ := strconv.ParseFloat("1", 64)

	jsonStrTest, _ := ToJson("John Doe")
	jsonIntTest, _ := ToJson(flt)

	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{"Should convert to string", args{value: jsonStrTest}, `John Doe`},
		{"Should convert to int", args{value: jsonIntTest}, flt}, // 1.0. IDK why, but the number from JSON unmarshal type is float
		{"Should convert to float", args{value: "1.1"}, 1.1},
		{"Should convert to bool", args{value: "true"}, true},
		{"Should convert to array", args{value: `["a","b","c"]`}, []interface{}{"a", "b", "c"}},
		{"Should convert to object", args{value: `{"name":"John Doe"}`}, map[string]interface{}{"name": "John Doe"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := ToInterface(tt.args.value); !assert.Equal(t, tt.want, got) {
				t.Errorf("ToInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToJson(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should convert to json object", args{value: map[string]interface{}{"name": "John Doe"}}, `{"name":"John Doe"}`},
		{"Should convert to json array", args{value: []string{"John Doe"}}, `["John Doe"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := ToJson(tt.args.value); got != tt.want {
				t.Errorf("ToJson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_connect(t *testing.T) {
	type args struct {
		host     string
		password string
		database int
	}
	tests := []struct {
		name string
		args args
		want *redis.Client
	}{
		{
			"Should connect to redis",
			args{host: "localhost:6379", password: "", database: 0},
			redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connect(tt.args.host, tt.args.password, tt.args.database); !assert.IsTypef(t, tt.want, got, "connect() = %v, want %v", got, tt.want) {
				t.Errorf("connect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getKeyWithPrefix(t *testing.T) {
	type args struct {
		c     *RedisConfig
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should give prefix and value", args{&RedisConfig{Prefix: "prefix"}, "value"}, "prefix:value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getKeyWithPrefix(tt.args.c, tt.args.value); got != tt.want {
				t.Errorf("getKeyWithPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getenv(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Log(err)
	}

	log.Printf("env: %v", os.Getenv("REDIS_HOST"))

	type args struct {
		key      string
		fallback string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"REDIS_HOST is set", args{"REDIS_HOST", "localhost"}, "host.docker.internal"},
		{"REDIS_HOSTS is not set", args{"REDIS_HOSTS", "localhost"}, "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getenv(tt.args.key, tt.args.fallback); got != tt.want {
				t.Errorf("getenv() = %v, want %v", got, tt.want)
			}
		})
	}
}
