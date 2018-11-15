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
	"time"

	"github.com/garyburd/redigo/redis"
)

// Constants for timeouts.
const (
	ConnectTimeout = 10 * time.Second
	ReadTimeout    = 10 * time.Second
	WriteTimeout   = 10 * time.Second
)

type redisStore struct {
	addr   string
	ttlsec int
}

// NewRedisStore returns a new instance of KVStore backed by a redis server.
// In this constructor we try to open a connection to redis. If that attempt fails
// we return an error. If it succeeds we just close that connection.
func NewRedisStore(addr string, ttlsec int) (KVStore, error) {
	conn, err := redis.DialTimeout("tcp", addr, ConnectTimeout, ReadTimeout, WriteTimeout)
	if err != nil {
		return nil, err
	}
	return &redisStore{
		addr:   addr,
		ttlsec: ttlsec,
	}, conn.Close()
}

func (store *redisStore) Get(key string) (string, error) {
	conn, err := redis.DialTimeout("tcp", store.addr, ConnectTimeout, ReadTimeout, WriteTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	reply, err := redis.String(conn.Do("GET", key))
	if err == redis.ErrNil {
		return "", nil
	}
	return reply, err
}

func (store *redisStore) Put(key, value string) error {
	conn, err := redis.DialTimeout("tcp", store.addr, ConnectTimeout, ReadTimeout, WriteTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Do("SETEX", key, store.ttlsec, value)
	return err
}

func (store *redisStore) Cleanup() error { return nil }
