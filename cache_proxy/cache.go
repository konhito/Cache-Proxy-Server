package cache_proxy

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var (
	ctx    = context.Background()
	client *redis.Client
)

func initRedis() {
	opt, err := redis.ParseURL("redis://default@redis:6379")
	if err != nil {
		panic(err)
	}

	client = redis.NewClient(opt)
	err = client.Ping(ctx).Err()
	if err != nil {
		panic(fmt.Sprintf("Redis ping failed: %v", err))
	}

	fmt.Println("Redis initialized successfully")
}

func GetClient() *redis.Client {
	return client
}
