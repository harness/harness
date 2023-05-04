package cron

import (
	"context"
	"errors"
	"fmt"

	cron "github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

const (
	//Format: seconds minute(0-59) hour(0-23) day of month(1-31) month(1-12) day of week(0-6)
	Hourly      = "0 0 * * * *" // once an hour at minute 0
	Nightly     = "0 0 0 * * *" // once a day at midnight
	Weekly      = "0 0 0 * * 0" // once a week on Sun midnight
	Monthly     = "0 0 0 1 * *" // once a month on the first day of the month
	EverySecond = "* * * * * *" // every second (for testing)
)

var ErrFatal = errors.New("fatal error occured")

type CronManager struct {
	c      *cron.Cron
	ctx    context.Context
	cancel context.CancelFunc
	fatal  chan error
}

// options could be location, logger, etc.
func NewCronManager() *CronManager {
	return &CronManager{
		c:     cron.New(cron.WithSeconds()),
		fatal: make(chan error),
	}
}

// add a new func to cron job
func (c *CronManager) NewCronTask(sepc string, job func(ctx context.Context) error) error {
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
func (c *CronManager) Run(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	var err error
	go func() {
		select {
		case <-ctx.Done():
			err = fmt.Errorf("context done: %w", ctx.Err())
		case fErr := <-c.fatal:
			err = fmt.Errorf("fatal error occured: %w", fErr)
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
