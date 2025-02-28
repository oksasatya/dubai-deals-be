package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

var Ctx = context.Background()

// NewRedisClient function to initialize Redis client
func NewRedisClient() *redis.Client {
	redisAddr := os.Getenv("REDIS_URL")
	fmt.Println("Connecting to Redis at:", redisAddr)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Cek koneksi
	_, err := rdb.Ping(Ctx).Result()
	if err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}
	logrus.Infof("Successfully connected to Redis")
	return rdb
}

// StoreTokenInRedis function to store token in Redis
func StoreTokenInRedis(ctx context.Context, token, userID, role string, ttlHours int) error {
	rdb := NewRedisClient()

	tokenData := map[string]interface{}{
		"userID": userID,
		"role":   role,
	}

	tokenJSON, err := json.Marshal(tokenData)
	if err != nil {
		return err
	}

	err = rdb.Set(ctx, token, string(tokenJSON), time.Duration(ttlHours)*time.Hour).Err()
	if err != nil {
		return err
	}

	return nil
}
func BlacklistToken(token string, expiration time.Duration) error {
	rdb := NewRedisClient()
	ctx := context.Background()

	return rdb.Set(ctx, "blacklist:"+token, "blacklisted", expiration).Err()
}
