// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// Package Docker is DEPRECATED.  DO NOT UPDATE.

package docker

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/drone/drone/core"
	"github.com/drone/drone/operator/runner"
)

var _ core.Scheduler = (*Scheduler)(nil)

// Scheduler is a Docker scheduler.
type Scheduler struct {
	sync.Mutex

	pending chan int64
	running map[int64]context.CancelFunc
	runner  *runner.Runner
}

// New returns a new Docker scheduler.
func New() *Scheduler {
	return NewRunner(nil)
}

// NewRunner returns a new Docker scheduler.
func NewRunner(runner *runner.Runner) *Scheduler {
	return &Scheduler{
		pending: make(chan int64, 100),
		running: make(map[int64]context.CancelFunc),
		runner:  runner,
	}
}

// SetRunner sets the build runner.
func (s *Scheduler) SetRunner(runner *runner.Runner) {
	s.runner = runner
}

// Start starts the scheduler.
func (s *Scheduler) Start(ctx context.Context, cap int) error {
	var g errgroup.Group
	for i := 0; i < cap; i++ {
		g.Go(func() error {
			return s.start(ctx)
		})
	}
	return g.Wait()
}

func (s *Scheduler) start(ctx context.Context) error {
	for {
		select {
		case id := <-s.pending:
			s.run(ctx, id)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Scheduler) run(ctx context.Context, id int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.Lock()
	s.running[id] = cancel
	s.Unlock()

	err := s.runner.Run(ctx, id)

	s.Lock()
	delete(s.running, id)
	s.Unlock()

	return err
}

// Schedule schedules the stage for execution.
func (s *Scheduler) Schedule(ctx context.Context, stage *core.Stage) error {
	select {
	case s.pending <- stage.ID:
		return nil
	default:
		return errors.New("Internal build queue is full")
	}
}

// Cancel cancels scheduled or running jobs associated
// with the parent build ID.
func (s *Scheduler) Cancel(_ context.Context, id int64) error {
	s.Lock()
	cancel, ok := s.running[id]
	s.Unlock()
	if ok {
		cancel()
	}
	return nil
}

// Stats returns stats specific to the scheduler.
func (s *Scheduler) Stats(_ context.Context) (interface{}, error) {
	s.Lock()
	stats := &status{
		Pending: len(s.pending),
		Running: len(s.running),
	}
	s.Unlock()
	return stats, nil
}

func (s *Scheduler) Request(context.Context, core.Filter) (*core.Stage, error) {
	return nil, nil
}

func (s *Scheduler) Cancelled(context.Context, int64) (bool, error) {
	return false, nil
}

type status struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
}
