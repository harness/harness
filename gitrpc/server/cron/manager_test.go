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

package cron

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func run(ctx context.Context, cmngr *Manager) chan error {
	cron := make(chan error)
	go func() {
		cron <- cmngr.Run(ctx)
	}()
	return cron
}

func TestCronManagerFatalErr(t *testing.T) {
	cmngr := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = cmngr.NewCronTask(EverySecond, func(ctx context.Context) error {
		return fmt.Errorf("inner: %w", ErrFatal)
	})
	select {
	case ferr := <-run(ctx, cmngr):
		if ferr == nil {
			t.Error("Cronmanager failed to receive fatal error")
		}
	case <-time.After(2 * time.Second):
		t.Error("Cronmanager failed to stop after a fatal error")
	}
}

func TestCronManagerNonFatalErr(t *testing.T) {
	cmngr := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = cmngr.NewCronTask(EverySecond, func(ctx context.Context) error {
		return errors.New("dummy error")
	})
	select {
	case ferr := <-run(ctx, cmngr):
		if ferr != nil {
			t.Error("Cronmanager failed at a non fatal error")
		}
	case <-time.After(1500 * time.Microsecond):
		// cron manager should keep running
	}
}
func TestCronManagerNewTask(t *testing.T) {
	cmngr := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := 0
	// add a task
	_ = cmngr.NewCronTask(EverySecond, func(ctx context.Context) error {
		a = 1
		return nil
	})

	select {
	case cerr := <-run(ctx, cmngr):
		if cerr != nil {
			t.Error("Cronmanager failed at Run:", cerr)
		}
	case <-time.After(1500 * time.Millisecond):
		if a != 1 {
			t.Error("Cronmanager failed to run the task")
		}
	}
}

func TestCronManagerStopOnCtxCancel(t *testing.T) {
	cmngr := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = cmngr.NewCronTask(EverySecond, func(ctx context.Context) error {
		cancel()
		return nil
	})
	err := cmngr.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Error("Cronmanager failed to stop after ctx got canceled ", err)
	}
}

func TestCronManagerStopOnCtxTimeout(t *testing.T) {
	cmngr := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = cmngr.NewCronTask(EverySecond, func(ctx context.Context) error {
		time.Sleep(5 * time.Second)
		return nil
	})
	err := cmngr.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("Cronmanager failed to stop after ctx timeout", err)
	}
}
