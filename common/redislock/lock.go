/*
 * Copyright © 2021 ZkBNB Protocol
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package redislock

import (
	"github.com/bnb-chain/zkbnb/types"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	LockExpiryTime = 10 // seconds
	RetryInterval  = 500 * time.Millisecond
	MaxRetryTimes  = 3
)

func GetRedisLockByKey(conn *redis.Redis, keyLock string) (redisLock *redis.RedisLock) {
	// get lock
	redisLock = redis.NewRedisLock(conn, keyLock)
	// set expiry time
	redisLock.SetExpire(LockExpiryTime)
	return redisLock
}

func GetRedisLock(conn *redis.Redis, keyLock string, expireSeconds int) (redisLock *redis.RedisLock) {
	// get lock
	redisLock = redis.NewRedisLock(conn, keyLock)
	// set expiry time
	redisLock.SetExpire(expireSeconds)
	return redisLock
}

func TryAcquireLock(redisLock *redis.RedisLock) (err error) {
	// lock
	ok, err := redisLock.Acquire()
	if err != nil {
		return err
	}
	// re-try for three times
	if !ok {
		ticker := time.NewTicker(RetryInterval)
		defer ticker.Stop()
		count := 0
		for {
			if count > MaxRetryTimes {
				return types.AppErrLockUsed
			}
			ok, err = redisLock.Acquire()
			if err != nil {
				return err
			}
			if ok {
				break
			}
			count++
			<-ticker.C
		}
	}
	return nil
}
