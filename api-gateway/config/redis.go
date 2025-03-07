package config

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

var Ctx = context.Background()
var RedisClient *redis.Client

// InitRedis function to initialize Redis client
func InitRedis() {
	redisAddr := os.Getenv("REDIS_URL")

	if redisAddr == "" {
		logrus.Fatal("REDIS_URL is required")
	}

	opt, err := redis.ParseURL(redisAddr)
	if err != nil {
		logrus.Fatalf("Failed to parse Redis URL: %v", err)
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: opt.Addr,
		DB:   0,
	})

	// Test Redis connection
	_, err = RedisClient.Ping(Ctx).Result()
	if err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}

	logrus.Infof("Successfully connected to Redis at %s", redisAddr)
}

// StoreTokenInRedis function to store token in Redis
func StoreTokenInRedis(ctx context.Context, token, userID, role string, ttlHours int) error {
	if RedisClient == nil {
		logrus.Errorf("Redis client is not initialized")
		return errors.New("redis client is not initialized")
	}

	tokenData := map[string]interface{}{
		"userID": userID,
		"role":   role,
	}

	tokenJSON, err := json.Marshal(tokenData)
	if err != nil {
		return err
	}

	err = RedisClient.Set(ctx, token, string(tokenJSON), time.Duration(ttlHours)*time.Hour).Err()
	if err != nil {
		logrus.Errorf("Failed to store token in Redis: %v", err)
		return err
	}

	return nil
}

// BlacklistToken function to blacklist token
func BlacklistToken(token string, expiration time.Duration) error {
	if RedisClient == nil {
		logrus.Errorf("Redis client is nil")
		return redis.Nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), expiration)
	defer cancel()

	err := RedisClient.Set(ctx, "blacklist:"+token, "blacklisted", expiration).Err()
	if err != nil {
		logrus.Errorf("Failed to blacklist token: %v", err)
		return err
	}
	logrus.Infof("Token blacklisted for %v", expiration)
	return nil
}

// IsBlacklistedToken function to check if token is blacklisted
func IsBlacklistedToken(token string) bool {
	if RedisClient == nil {
		logrus.Errorf("Redis client is nil")
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	blacklisted, err := RedisClient.Get(ctx, "blacklist:"+token).Result()
	if errors.Is(err, redis.Nil) {
		return false
	} else if err != nil {
		logrus.Errorf("Failed to check blacklist token: %v", err)
		return false
	}
	return blacklisted == "blacklisted"
}
