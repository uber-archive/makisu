//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

const (
	MaxRetires   = 3
	DialTimeout  = 10 * time.Second
	ReadTimeout  = 10 * time.Second
	WriteTimeout = 10 * time.Second
)

type redisStore struct {
	cli *redis.Client
	ttl time.Duration
}

// NewRedisStore returns a new instance of KVStore backed by a redis server.
// In this constructor we try to open a connection to redis. If that attempt fails
// we return an error. If it succeeds we just close that connection.
func NewRedisStore(addr string, ttl time.Duration) (KVStore, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:         addr,
		MaxRetries:   MaxRetires,
		DialTimeout:  DialTimeout,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
	})
	if _, err := cli.Ping().Result(); err != nil {
		return nil, err
	}
	return &redisStore{
		cli: cli,
		ttl: ttl,
	}, nil
}

func (store *redisStore) Get(key string) (string, error) {
	v, err := store.cli.Get(key).Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("redis get key: %s", err)
	}
	return v, err
}

func (store *redisStore) Put(key, value string) error {
	if _, err := store.cli.Set(key, value, store.ttl).Result(); err != nil {
		return fmt.Errorf("redis set key: %s", err)
	}
	return nil
}

func (store *redisStore) Cleanup() error { return nil }
