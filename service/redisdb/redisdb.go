// Copyright 2021 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redisdb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"

	"github.com/drone/drone/cmd/drone-server/config"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func New(config config.Config) (srv RedisDB, err error) {
	var options *redis.Options

	if config.Redis.ConnectionString != "" {
		options, err = redis.ParseURL(config.Redis.ConnectionString)
		if err != nil {
			return
		}
	} else if config.Redis.Addr != "" {
		options = &redis.Options{
			Addr:     config.Redis.Addr,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		}
	} else {
		return
	}

	rdb := redis.NewClient(options)

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		err = fmt.Errorf("redis not accessibe: %w", err)
		return
	}

	rs := redsync.New(goredis.NewPool(rdb))

	srv = redisService{
		rdb:      rdb,
		mutexGen: rs,
	}

	return
}

type RedisDB interface {
	Client() redis.Cmdable
	Subscribe(ctx context.Context, channelName string, channelSize int, proc PubSubProcessor)
	NewMutex(name string, expiry time.Duration) LockErr
}

type redisService struct {
	rdb      *redis.Client
	mutexGen *redsync.Redsync
}

// Client exposes redis.Cmdable interface
func (r redisService) Client() redis.Cmdable {
	return r.rdb
}

type PubSubProcessor interface {
	ProcessMessage(s string)
	ProcessError(err error)
}

var backoffDurations = []time.Duration{
	0, time.Second, 3 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second,
}

// Subscribe subscribes to a redis pub-sub channel. The messages are processed with the supplied PubSubProcessor.
// In case of en error the function will automatically reconnect with an increasing back of delay.
// The only way to exit this function is to terminate or expire the supplied context.
func (r redisService) Subscribe(ctx context.Context, channelName string, channelSize int, proc PubSubProcessor) {
	var connectTry int
	for {
		err := func() (err error) {
			defer func() {
				// panic recovery because external PubSubProcessor methods might cause panics.
				if p := recover(); p != nil {
					err = fmt.Errorf("redis pubsub: panic: %v", p)
				}
			}()

			var options []redis.ChannelOption

			if channelSize > 1 {
				options = append(options, redis.WithChannelSize(channelSize))
			}

			pubsub := r.rdb.Subscribe(ctx, channelName)
			ch := pubsub.Channel(options...)

			defer func() {
				_ = pubsub.Close()
			}()

			// make sure the connection is successful
			err = pubsub.Ping(ctx)
			if err != nil {
				return
			}

			connectTry = 0 // successfully connected, reset the counter

			logrus.
				WithField("try", connectTry+1).
				WithField("channel", channelName).
				Trace("redis pubsub: subscribed")

			for {
				select {
				case m, ok := <-ch:
					if !ok {
						err = fmt.Errorf("redis pubsub: channel=%s closed", channelName)
						return
					}

					proc.ProcessMessage(m.Payload)

				case <-ctx.Done():
					err = ctx.Err()
					return
				}
			}
		}()
		if err == nil {
			// should not happen, the function should always exit with an error
			continue
		}

		proc.ProcessError(err)

		if err == context.Canceled || err == context.DeadlineExceeded {
			logrus.
				WithField("channel", channelName).
				Trace("redis pubsub: finished")
			return
		}

		dur := backoffDurations[connectTry]

		logrus.
			WithError(err).
			WithField("try", connectTry+1).
			WithField("pause", dur.String()).
			WithField("channel", channelName).
			Error("redis pubsub: connection failed, reconnecting")

		time.Sleep(dur)

		if connectTry < len(backoffDurations)-1 {
			connectTry++
		}
	}
}

func (r redisService) NewMutex(name string, expiry time.Duration) LockErr {
	var options []redsync.Option
	if expiry > 0 {
		options = append(options, redsync.WithExpiry(expiry))
	}

	return r.mutexGen.NewMutex(name, options...)
}
