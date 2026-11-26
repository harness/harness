// Copyright 2026 Harness, Inc.
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

package job

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/store"
)

var errInfBoom = errors.New("boom")

// infFakeStore is an in-memory implementation of Store with optional error injection.
type infFakeStore struct {
	mx   sync.Mutex
	jobs map[string]*Job

	errFind        error
	errCreate      error
	errUpsert      error
	errUpdateExec  error
	errUpdateProg  error
	errCountRun    error
	errListReady   error
	errListOverdue error
	errNextTime    error
	errDeleteOld   error
	errDeleteGroup error
	errDeleteUID   error
	errListGroup   error

	deletedOld int64
}

func infNewFakeStore(jobs ...*Job) *infFakeStore {
	s := &infFakeStore{jobs: map[string]*Job{}}
	for _, j := range jobs {
		cp := *j
		s.jobs[j.UID] = &cp
	}
	return s
}

func (s *infFakeStore) get(uid string) (Job, bool) {
	s.mx.Lock()
	defer s.mx.Unlock()
	j, ok := s.jobs[uid]
	if !ok {
		return Job{}, false
	}
	return *j, true
}

func (s *infFakeStore) Find(_ context.Context, uid string) (*Job, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errFind != nil {
		return nil, s.errFind
	}
	j, ok := s.jobs[uid]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	cp := *j
	return &cp, nil
}

func (s *infFakeStore) ListByGroupID(_ context.Context, groupID string) ([]*Job, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errListGroup != nil {
		return nil, s.errListGroup
	}
	var res []*Job
	for _, j := range s.jobs {
		if j.GroupID == groupID {
			cp := *j
			res = append(res, &cp)
		}
	}
	sort.Slice(res, func(a, b int) bool { return res[a].UID < res[b].UID })
	return res, nil
}

func (s *infFakeStore) DeleteByGroupID(_ context.Context, groupID string) (int64, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errDeleteGroup != nil {
		return 0, s.errDeleteGroup
	}
	var n int64
	for uid, j := range s.jobs {
		if j.GroupID == groupID {
			delete(s.jobs, uid)
			n++
		}
	}
	return n, nil
}

func (s *infFakeStore) Create(_ context.Context, job *Job) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errCreate != nil {
		return s.errCreate
	}
	cp := *job
	s.jobs[job.UID] = &cp
	return nil
}

func (s *infFakeStore) Upsert(_ context.Context, job *Job) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errUpsert != nil {
		return s.errUpsert
	}
	cp := *job
	s.jobs[job.UID] = &cp
	return nil
}

func (s *infFakeStore) UpdateDefinition(ctx context.Context, job *Job) error {
	return s.UpdateExecution(ctx, job)
}

func (s *infFakeStore) UpdateExecution(_ context.Context, job *Job) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errUpdateExec != nil {
		return s.errUpdateExec
	}
	cp := *job
	s.jobs[job.UID] = &cp
	return nil
}

func (s *infFakeStore) UpdateProgress(_ context.Context, job *Job) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errUpdateProg != nil {
		return s.errUpdateProg
	}
	j, ok := s.jobs[job.UID]
	if !ok {
		return store.ErrResourceNotFound
	}
	j.RunProgress = job.RunProgress
	j.Result = job.Result
	return nil
}

func (s *infFakeStore) CountRunning(context.Context) (int, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errCountRun != nil {
		return 0, s.errCountRun
	}
	n := 0
	for _, j := range s.jobs {
		if j.State == JobStateRunning {
			n++
		}
	}
	return n, nil
}

func (s *infFakeStore) ListReady(_ context.Context, now time.Time, limit int) ([]*Job, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errListReady != nil {
		return nil, s.errListReady
	}
	var res []*Job
	for _, j := range s.jobs {
		if j.State == JobStateScheduled && j.Scheduled <= now.UnixMilli() {
			cp := *j
			res = append(res, &cp)
		}
	}
	sort.Slice(res, func(a, b int) bool { return res[a].UID < res[b].UID })
	if len(res) > limit {
		res = res[:limit]
	}
	return res, nil
}

func (s *infFakeStore) ListDeadlineExceeded(_ context.Context, now time.Time) ([]*Job, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errListOverdue != nil {
		return nil, s.errListOverdue
	}
	var res []*Job
	for _, j := range s.jobs {
		if j.State == JobStateRunning && j.RunDeadline < now.UnixMilli() {
			cp := *j
			res = append(res, &cp)
		}
	}
	sort.Slice(res, func(a, b int) bool { return res[a].UID < res[b].UID })
	return res, nil
}

func (s *infFakeStore) NextScheduledTime(_ context.Context, now time.Time) (time.Time, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errNextTime != nil {
		return time.Time{}, s.errNextTime
	}
	var next int64
	for _, j := range s.jobs {
		if j.State == JobStateScheduled && j.Scheduled > now.UnixMilli() && (next == 0 || j.Scheduled < next) {
			next = j.Scheduled
		}
	}
	if next == 0 {
		return time.Time{}, nil
	}
	return time.UnixMilli(next), nil
}

func (s *infFakeStore) DeleteOld(context.Context, time.Time) (int64, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errDeleteOld != nil {
		return 0, s.errDeleteOld
	}
	return s.deletedOld, nil
}

func (s *infFakeStore) DeleteByUID(_ context.Context, uid string) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.errDeleteUID != nil {
		return s.errDeleteUID
	}
	delete(s.jobs, uid)
	return nil
}

// infFakeMutexManager hands out mutexes that can be configured to fail.
type infFakeMutexManager struct {
	errNew    error
	errLock   error
	errUnlock error
}

func (m *infFakeMutexManager) NewMutex(key string, _ ...lock.Option) (lock.Mutex, error) {
	if m.errNew != nil {
		return nil, m.errNew
	}
	return &infFakeMutex{key: key, mgr: m}, nil
}

type infFakeMutex struct {
	key string
	mgr *infFakeMutexManager
}

func (m *infFakeMutex) Key() string { return m.key }

func (m *infFakeMutex) Lock(context.Context) error { return m.mgr.errLock }

func (m *infFakeMutex) Unlock(context.Context) error { return m.mgr.errUnlock }

// infFakePubSub records publishes and ignores subscriptions.
type infFakePubSub struct {
	mx          sync.Mutex
	published   map[string][][]byte
	errPublish  error
	subscribed  []string
	closedCount int
}

func infNewFakePubSub() *infFakePubSub {
	return &infFakePubSub{published: map[string][][]byte{}}
}

func (p *infFakePubSub) Publish(_ context.Context, topic string, payload []byte, _ ...pubsub.PublishOption) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	if p.errPublish != nil {
		return p.errPublish
	}
	p.published[topic] = append(p.published[topic], payload)
	return nil
}

func (p *infFakePubSub) Subscribe(
	_ context.Context,
	topic string,
	_ func(payload []byte) error,
	_ ...pubsub.SubscribeOption,
) pubsub.Consumer {
	p.mx.Lock()
	defer p.mx.Unlock()
	p.subscribed = append(p.subscribed, topic)
	return &infFakeConsumer{ps: p}
}

func (p *infFakePubSub) count(topic string) int {
	p.mx.Lock()
	defer p.mx.Unlock()
	return len(p.published[topic])
}

type infFakeConsumer struct {
	ps *infFakePubSub
}

func (c *infFakeConsumer) Subscribe(context.Context, ...string) error   { return nil }
func (c *infFakeConsumer) Unsubscribe(context.Context, ...string) error { return nil }
func (c *infFakeConsumer) Close() error {
	c.ps.mx.Lock()
	defer c.ps.mx.Unlock()
	c.ps.closedCount++
	return errInfBoom // exercises the error branch of the deferred close in Run
}

// infHandlerFunc adapts a function to the Handler interface.
type infHandlerFunc func(ctx context.Context, input string, fn ProgressReporter) (string, error)

func (f infHandlerFunc) Handle(ctx context.Context, input string, fn ProgressReporter) (string, error) {
	return f(ctx, input, fn)
}

// infNewScheduler builds a Scheduler wired with fakes; signal/done are initialised so that
// scheduling helpers never block on nil channels.
func infNewScheduler(st Store, mm lock.MutexManager, ps *infFakePubSub, maxRunning int) (*Scheduler, *Executor) {
	exec := NewExecutor(st, ps)
	s, err := NewScheduler(st, exec, mm, ps, "inst-1", maxRunning, time.Hour)
	if err != nil {
		panic(err)
	}
	s.signal = make(chan time.Time, 16)
	s.done = make(chan struct{})
	return s, exec
}
