package library

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"time"
	"ussd-wrapper/library/logger"
)

func GetRedisKey(conn *redis.Client, key string) (string, error) {

	var data string
	data, err := conn.Get(key).Result()
	if err != nil {

		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}

	return data, err

}

func SetRedisKey(conn *redis.Client, key string, value string) error {

	_, err := conn.Set(key, value, time.Second*time.Duration(0)).Result()
	if err != nil {

		v := value

		if len(v) > 15 {

			v = v[0:12] + "..."
		}

		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}
	return err
}

func SetRedisKeyWithExpiry(ctx context.Context, conn *redis.Client, key string, value string, seconds int) error {

	_, err := conn.Set(key, value, time.Second*time.Duration(seconds)).Result()
	if err != nil {

		v := value

		if len(v) > 15 {

			v = v[0:12] + "..."
		}
		logger.WithCtx(ctx).Infof("error saving redisKey %s error %s", key, err.Error())
		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}

	return err
}

func IncRedisKey(conn *redis.Client, key string) (int64, error) {

	var data int64
	data, err := conn.Incr(key).Result()

	if err != nil {

		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}

	return data, err
}
func DecRedisKey(conn *redis.Client, key string) (int64, error) {

	var data int64
	data, err := conn.Decr(key).Result()

	if err != nil {

		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}

	return data, err
}

func DeleteRedisKey(ctx context.Context, conn *redis.Client, key string) error {
	// Attempt to delete the key from Redis
	result, err := conn.Del(key).Result()
	if err != nil {
		return fmt.Errorf("error deleting key %s: %v", key, err)
	}

	// Check if the key was actually deleted
	if result == 0 {
		logger.WithCtx(ctx).Infof("key %s was not found in Redis", key)
	} else {
		logger.WithCtx(ctx).Infof("key %s deleted successfully", key)
	}

	return nil
}
