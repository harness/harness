// Copyright 2019 Drone IO, Inc.
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

package queue

import (
	"context"
	"sync"
	"time"

	"github.com/drone/drone/core"
)

type queue struct {
	sync.Mutex

	ready    chan struct{}
	paused   bool
	interval time.Duration
	store    core.StageStore
	workers  map[*worker]struct{}
	ctx      context.Context
}

// newQueue returns a new Queue backed by the build datastore.
func newQueue(store core.StageStore) *queue {
	q := &queue{
		store:    store,
		ready:    make(chan struct{}, 1),
		workers:  map[*worker]struct{}{},
		interval: time.Minute,
		ctx:      context.Background(),
	}
	go q.start()
	return q
}

func (q *queue) Schedule(ctx context.Context, stage *core.Stage) error {
	select {
	case q.ready <- struct{}{}:
	default:
	}
	return nil
}

func (q *queue) Pause(ctx context.Context) error {
	q.Lock()
	q.paused = true
	q.Unlock()
	return nil
}

func (q *queue) Paused(ctx context.Context) (bool, error) {
	q.Lock()
	paused := q.paused
	q.Unlock()
	return paused, nil
}

func (q *queue) Resume(ctx context.Context) error {
	q.Lock()
	q.paused = false
	q.Unlock()

	select {
	case q.ready <- struct{}{}:
	default:
	}
	return nil
}

func (q *queue) Request(ctx context.Context, params core.Filter) (*core.Stage, error) {
	w := &worker{
		kind:    params.Kind,
		typ:     params.Type,
		os:      params.OS,
		arch:    params.Arch,
		kernel:  params.Kernel,
		variant: params.Variant,
		labels:  params.Labels,
		channel: make(chan *core.Stage),
	}
	q.Lock()
	q.workers[w] = struct{}{}
	q.Unlock()

	select {
	case q.ready <- struct{}{}:
	default:
	}

	select {
	case <-ctx.Done():
		q.Lock()
		delete(q.workers, w)
		q.Unlock()
		return nil, ctx.Err()
	case b := <-w.channel:
		return b, nil
	}
}

func (q *queue) signal(ctx context.Context) error {
	q.Lock()
	count := len(q.workers)
	pause := q.paused
	q.Unlock()
	if pause {
		return nil
	}
	if count == 0 {
		return nil
	}
	items, err := q.store.ListIncomplete(ctx)
	if err != nil {
		return err
	}

	q.Lock()
	defer q.Unlock()
	for _, item := range items {
		if item.Status == core.StatusRunning {
			continue
		}
		if item.Machine != "" {
			continue
		}

		// if the stage defines concurrency limits we
		// need to make sure those limits are not exceeded
		// before proceeding.
		if withinLimits(item, items) == false {
			continue
		}

	loop:
		for w := range q.workers {
			// the worker must match the resource kind and type
			if !matchResource(w.kind, w.typ, item.Kind, item.Type) {
				continue
			}

			if w.os != "" || w.arch != "" || w.variant != "" || w.kernel != "" {
				// the worker is platform-specific. check to ensure
				// the queue item matches the worker platform.
				if w.os != item.OS {
					continue
				}
				if w.arch != item.Arch {
					continue
				}
				// if the pipeline defines a variant it must match
				// the worker variant (e.g. arm6, arm7, etc).
				if item.Variant != "" && item.Variant != w.variant {
					continue
				}
				// if the pipeline defines a kernel version it must match
				// the worker kernel version (e.g. 1709, 1803).
				if item.Kernel != "" && item.Kernel != w.kernel {
					continue
				}
			}

			if len(item.Labels) > 0 || len(w.labels) > 0 {
				if !checkLabels(item.Labels, w.labels) {
					continue
				}
			}

			// // the queue has 60 seconds to ack the item, otherwise
			// // it is eligible for processing by another worker.
			// // item.Expires = time.Now().Add(time.Minute).Unix()
			// err := q.store.Update(ctx, item)

			// if err != nil {
			// 	log.Ctx(ctx).Warn().
			// 		Err(err).
			// 		Int64("build_id", item.BuildID).
			// 		Int64("stage_id", item.ID).
			// 		Msg("cannot update queue item")
			// 	continue
			// }
			select {
			case w.channel <- item:
				delete(q.workers, w)
				break loop
			}
		}
	}
	return nil
}

func (q *queue) start() error {
	for {
		select {
		case <-q.ctx.Done():
			return q.ctx.Err()
		case <-q.ready:
			q.signal(q.ctx)
		case <-time.After(q.interval):
			q.signal(q.ctx)
		}
	}
}

type worker struct {
	kind    string
	typ     string
	os      string
	arch    string
	kernel  string
	variant string
	labels  map[string]string
	channel chan *core.Stage
}

type counter struct {
	counts map[string]int
}

func checkLabels(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func withinLimits(stage *core.Stage, siblings []*core.Stage) bool {
	if stage.Limit == 0 {
		return true
	}
	count := 0
	for _, sibling := range siblings {
		if sibling.RepoID != stage.RepoID {
			continue
		}
		if sibling.ID == stage.ID {
			continue
		}
		if sibling.Name != stage.Name {
			continue
		}
		if sibling.ID < stage.ID {
			count++
		}
	}
	return count < stage.Limit
}

// matchResource is a helper function that returns
func matchResource(kinda, typea, kindb, typeb string) bool {
	if kinda == "" {
		kinda = "pipeline"
	}
	if kindb == "" {
		kindb = "pipeline"
	}
	if typea == "" {
		typea = "docker"
	}
	if typeb == "" {
		typeb = "docker"
	}
	return kinda == kindb && typea == typeb
}
