// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	cron "github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

// Format: seconds minute(0-59) hour(0-23) day of month(1-31) month(1-12) day of week(0-6).
const (
	Hourly      = "0 0 * * * *" // once an hour at minute 0
	Nightly     = "0 0 0 * * *" // once a day at midnight
	Weekly      = "0 0 0 * * 0" // once a week on Sun midnight
	Monthly     = "0 0 0 1 * *" // once a month on the first day of the month
	EverySecond = "* * * * * *" // every second (for testing)
)

var ErrFatal = errors.New("fatal error occurred")

type Manager struct {
	c      *cron.Cron
	ctx    context.Context
	cancel context.CancelFunc
	fatal  chan error
}

// NewManager creates a cron manager.
func NewManager() *Manager {
	return &Manager{
		c:     cron.New(cron.WithSeconds()),
		fatal: make(chan error),
	}
}

// NewCronTask adds a new func to cron job.
func (c *Manager) NewCronTask(sepc string, job func(ctx context.Context) error) error {
	_, err := c.c.AddFunc(sepc, func() {
		jerr := job(c.ctx)
		if jerr != nil { // check different severity of errors
			log.Ctx(c.ctx).Error().Err(jerr).Msg("gitrpc cron job failed")

			if errors.Is(jerr, ErrFatal) {
				c.fatal <- jerr
				return
			}
		}
	})
	if err != nil {
		return fmt.Errorf("gitrpc cron manager failed to add cron job function: %w", err)
	}
	return nil
}

// Run the cron scheduler, or no-op if already running.
func (c *Manager) Run(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	var err error
	go func() {
		select {
		case <-ctx.Done():
			err = fmt.Errorf("context done: %w", ctx.Err())
		case fErr := <-c.fatal:
			err = fmt.Errorf("fatal error occurred: %w", fErr)
		}

		// stop scheduling of new jobs.
		// NOTE: doesn't wait for running jobs, but c.Run() does, and we don't have to wait here
		_ = c.c.Stop()

		// cancel running jobs (redundant for ctx.Done(), but makes code simpler)
		c.cancel()
	}()

	c.c.Run()
	close(c.fatal)
	return err
}
