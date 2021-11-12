package cache

import (
	"context"
	"math/rand"
	"time"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

const (
	MUTEX_EXPIRY_SEC = 8
	SLEEP            = 200 * time.Millisecond

	// lock retry parameters
	NEW_MUTEX_TRIES       = 1000
	minRetryDelayMilliSec = 5
	maxRetryDelayMilliSec = 10
)

var ctx = context.Background()

type RedisDB struct {
	client *goredislib.Client
	rs     *redsync.Redsync
}

func NewRedisDB(address string, password string) *RedisDB {
	client := goredislib.NewClient(&goredislib.Options{
		Addr:     address,
		Password: password,
	})
	// TODO check connectivity
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)
	return &RedisDB{client, rs}
}

func (rdb *RedisDB) Set(key string, value string) error {
	return rdb.client.Set(ctx, key, value, 0).Err()
}

func (rdb *RedisDB) Get(key string) (string, error) {
	return rdb.client.Get(ctx, key).Result()
}

func (rdb *RedisDB) Del(key string) (int64, error) {
	return rdb.client.Del(ctx, key).Result()
}

func (rdb *RedisDB) GetNewMutex(key string) *redsync.Mutex {
	key = key + "-mutex"
	return rdb.rs.NewMutex(key,
		redsync.WithExpiry(MUTEX_EXPIRY_SEC*time.Second),
		redsync.WithTries(NEW_MUTEX_TRIES),
		redsync.WithRetryDelayFunc(customDelayFunc),
	)
}

func customDelayFunc(tries int) time.Duration {
	return time.Duration(rand.Intn(maxRetryDelayMilliSec-minRetryDelayMilliSec)+minRetryDelayMilliSec) * time.Millisecond
}
