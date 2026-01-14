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

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type queue struct {
	sync.Mutex
	globMx lock.Mutex

	ready    chan struct{}
	paused   bool
	interval time.Duration
	store    store.StageStore
	workers  map[*worker]struct{}
	ctx      context.Context
}

// newQueue returns a new Queue backed by the build datastore.
func newQueue(store store.StageStore, lock lock.MutexManager) (*queue, error) {
	const lockKey = "build_queue"
	mx, err := lock.NewMutex(lockKey)
	if err != nil {
		return nil, err
	}
	q := &queue{
		store:    store,
		globMx:   mx,
		ready:    make(chan struct{}, 1),
		workers:  map[*worker]struct{}{},
		interval: time.Minute,
		ctx:      context.Background(),
	}
	go func() {
		if err := q.start(); err != nil {
			log.Err(err).Msg("queue start failed")
		}
	}()

	return q, nil
}

func (q *queue) Schedule(_ context.Context, _ *types.Stage) error {
	select {
	case q.ready <- struct{}{}:
	default:
	}
	return nil
}

func (q *queue) Pause(_ context.Context) error {
	q.Lock()
	q.paused = true
	q.Unlock()
	return nil
}

func (q *queue) Request(ctx context.Context, params Filter) (*types.Stage, error) {
	w := &worker{
		kind:    params.Kind,
		typ:     params.Type,
		os:      params.OS,
		arch:    params.Arch,
		kernel:  params.Kernel,
		variant: params.Variant,
		labels:  params.Labels,
		channel: make(chan *types.Stage),
		done:    ctx.Done(),
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
		return nil, ctx.Err()
	case b := <-w.channel:
		return b, nil
	}
}

//nolint:gocognit // refactor if needed.
func (q *queue) signal(ctx context.Context) error {
	if err := q.globMx.Lock(ctx); err != nil {
		return err
	}
	defer func() {
		if err := q.globMx.Unlock(ctx); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to release global lock after signaling")
		}
	}()

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
		if item.Status == enum.CIStatusRunning {
			continue
		}
		if item.Machine != "" {
			continue
		}

		// if the stage defines concurrency limits we
		// need to make sure those limits are not exceeded
		// before proceeding.
		if !withinLimits(item, items) {
			continue
		}

		// if the system defines concurrency limits
		// per repository we need to make sure those limits
		// are not exceeded before proceeding.
		if shouldThrottle(item, items, item.LimitRepo) {
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

			select {
			case w.channel <- item:
			case <-w.done:
			}

			delete(q.workers, w)
			break loop
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
			if err := q.signal(q.ctx); err != nil {
				// don't return, only log error
				log.Ctx(q.ctx).Err(err).Msg("failed to signal on ready")
			}
		case <-time.After(q.interval):
			if err := q.signal(q.ctx); err != nil {
				// don't return, only log error
				log.Ctx(q.ctx).Err(err).Msg("failed to signal on interval")
			}
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
	channel chan *types.Stage
	done    <-chan struct{}
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

func withinLimits(stage *types.Stage, siblings []*types.Stage) bool {
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
		if sibling.ID < stage.ID ||
			sibling.Status == enum.CIStatusRunning {
			count++
		}
	}
	return count < stage.Limit
}

func shouldThrottle(stage *types.Stage, siblings []*types.Stage, limit int) bool {
	// if no throttle limit is defined (default) then
	// return false to indicate no throttling is needed.
	if limit == 0 {
		return false
	}
	// if the repository is running it is too late
	// to skip and we can exit
	if stage.Status == enum.CIStatusRunning {
		return false
	}

	count := 0
	// loop through running stages to count number of
	// running stages for the parent repository.
	for _, sibling := range siblings {
		// ignore stages from other repository.
		if sibling.RepoID != stage.RepoID {
			continue
		}
		// ignore this stage and stages that were
		// scheduled after this stage.
		if sibling.ID >= stage.ID {
			continue
		}
		count++
	}
	// if the count of running stages exceeds the
	// throttle limit return true.
	return count >= limit
}

// matchResource is a helper function that returns.
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
