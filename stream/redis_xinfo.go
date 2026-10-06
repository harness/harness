// Copyright 2023 Harness, Inc.
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

package stream

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// xInfoConsumer holds the fields of an XINFO CONSUMERS entry that we care about.
type xInfoConsumer struct {
	Name    string
	Pending int64
	Idle    int64
}

// readXInfoConsumers returns all consumers of the given group for the given stream.
//
// NOTE: The command is executed and parsed manually instead of using the client's XInfoConsumers
// method, because that method fails on any reply that doesn't consist of exactly the three fields
// name, pending and idle. Redis returns an additional inactive field since 7.2 and is free to add
// more fields at any time - so we parse the reply field by field and ignore what we don't know.
func readXInfoConsumers(
	ctx context.Context,
	rdb redis.UniversalClient,
	streamID string,
	groupName string,
) ([]xInfoConsumer, error) {
	// IMPORTANT: keep the args identical to what the client's XInfoConsumers method sends,
	// otherwise the command could get routed to a different node in a redis cluster.
	res, err := rdb.Do(ctx, "xinfo", "consumers", streamID, groupName).Result()
	if err != nil {
		return nil, err
	}

	return parseXInfoConsumers(res)
}

func parseXInfoConsumers(res interface{}) ([]xInfoConsumer, error) {
	entries, ok := res.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected an array in XINFO CONSUMERS reply but got %T", res)
	}

	consumers := make([]xInfoConsumer, 0, len(entries))
	for _, entry := range entries {
		consumer, err := parseXInfoConsumer(entry)
		if err != nil {
			return nil, err
		}

		consumers = append(consumers, consumer)
	}

	return consumers, nil
}

func parseXInfoConsumer(entry interface{}) (xInfoConsumer, error) {
	var consumer xInfoConsumer

	fields, ok := entry.([]interface{})
	if !ok {
		return consumer, fmt.Errorf("expected an array in XINFO CONSUMERS entry but got %T", entry)
	}
	if len(fields)%2 != 0 {
		return consumer, fmt.Errorf(
			"expected an even number of elements in XINFO CONSUMERS entry but got %d", len(fields))
	}

	for i := 0; i < len(fields); i += 2 {
		key, err := redisString(fields[i])
		if err != nil {
			return consumer, fmt.Errorf("failed to read field name in XINFO CONSUMERS entry: %w", err)
		}

		switch key {
		case "name":
			consumer.Name, err = redisString(fields[i+1])
		case "pending":
			consumer.Pending, err = redisInt64(fields[i+1])
		case "idle":
			consumer.Idle, err = redisInt64(fields[i+1])
		default:
			// ignore any field we don't use - redis keeps adding new ones
			continue
		}
		if err != nil {
			return consumer, fmt.Errorf("failed to read field '%s' in XINFO CONSUMERS entry: %w", key, err)
		}
	}

	if consumer.Name == "" {
		return consumer, errors.New("missing field 'name' in XINFO CONSUMERS entry")
	}

	return consumer, nil
}

func redisString(val interface{}) (string, error) {
	switch t := val.(type) {
	case string:
		return t, nil
	case []byte:
		return string(t), nil
	default:
		return "", fmt.Errorf("expected a string but got %T", val)
	}
}

func redisInt64(val interface{}) (int64, error) {
	switch t := val.(type) {
	case int64:
		return t, nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	case []byte:
		return strconv.ParseInt(string(t), 10, 64)
	default:
		return 0, fmt.Errorf("expected an integer but got %T", val)
	}
}
