package connections

import (
	"fmt"
	"github.com/go-redis/redis"
	"os"
	"strconv"
	"time"
)

func InitRedis() *redis.Client {

	host := os.Getenv("redis_host")
	port := os.Getenv("redis_port")
	db := os.Getenv("redis_database_number")
	auth := os.Getenv("redis_password")

	dbNumber, err := strconv.Atoi(db)
	if err != nil {

		dbNumber = 1
	}

	uri := fmt.Sprintf("redis://%s:%s", host, port)
	uri = fmt.Sprintf("%s:%s", host, port)

	opts := redis.Options{
		MinIdleConns: 10,
		IdleTimeout:  60 * time.Second,
		PoolSize:     1000,
		Addr:         uri,
		DB:           dbNumber, // use default DB
	}

	if len(auth) > 0 {

		opts.Password = auth
	}

	client := redis.NewClient(&opts)

	return client
}
