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

package usage

import "context"

type queue struct {
	ch chan Metric
}

func newQueue() *queue {
	return &queue{
		ch: make(chan Metric, 1024),
	}
}

func (q *queue) Add(ctx context.Context, payload Metric) {
	select {
	case <-ctx.Done():
		return
	case q.ch <- payload:
	default:
		// queue is full then wait in new go routine
		// until one of consumer read from channel,
		// we dont want to block caller goroutine
		go func() {
			q.ch <- payload
		}()
	}
}

func (q *queue) Pop(ctx context.Context) (*Metric, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case payload := <-q.ch:
		return &payload, nil
	}
}

func (q *queue) Close() {
	close(q.ch)
}

func (q *queue) Len() int {
	return len(q.ch)
}
