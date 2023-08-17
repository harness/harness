// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gorhill/cronexpr"
	"github.com/rs/zerolog/log"
)

// Scheduler controls execution of background jobs.
type Scheduler struct {
	// dependencies
	store         store.JobStore
	executor      *Executor
	mxManager     lock.MutexManager
	pubsubService pubsub.PubSub

	// configuration fields
	instanceID     string
	maxRunning     int
	purgeMinOldAge time.Duration

	// synchronization stuff
	signal       chan time.Time
	done         chan struct{}
	wgRunning    sync.WaitGroup
	cancelJobMx  sync.Mutex
	cancelJobMap map[string]context.CancelFunc
}

func NewScheduler(
	jobStore store.JobStore,
	executor *Executor,
	mxManager lock.MutexManager,
	pubsubService pubsub.PubSub,
	instanceID string,
	maxRunning int,
	purgeMinOldAge time.Duration,
) (*Scheduler, error) {
	if maxRunning < 1 {
		maxRunning = 1
	}
	return &Scheduler{
		store:         jobStore,
		executor:      executor,
		mxManager:     mxManager,
		pubsubService: pubsubService,

		instanceID:     instanceID,
		maxRunning:     maxRunning,
		purgeMinOldAge: purgeMinOldAge,

		cancelJobMap: map[string]context.CancelFunc{},
	}, nil
}

// Run runs the background job scheduler.
// It's a blocking call. It blocks until the provided context is done.
func (s *Scheduler) Run(ctx context.Context) error {
	if s.done != nil {
		return errors.New("already started")
	}

	consumer := s.pubsubService.Subscribe(ctx, PubSubTopicCancelJob, s.handleCancelJob)
	defer func() {
		err := consumer.Close()
		if err != nil {
			log.Ctx(ctx).Err(err).
				Msg("job scheduler: failed to close pubsub cancel job consumer")
		}
	}()

	if err := s.createNecessaryJobs(ctx); err != nil {
		return fmt.Errorf("failed to create necessary jobs: %w", err)
	}

	if err := s.registerNecessaryJobs(); err != nil {
		return fmt.Errorf("failed to register scheduler's internal jobs: %w", err)
	}

	s.executor.finishRegistration()

	log.Ctx(ctx).Debug().Msg("job scheduler: starting")

	s.done = make(chan struct{})
	defer close(s.done)

	s.signal = make(chan time.Time, 1)

	timer := newSchedulerTimer()
	defer timer.Stop()

	for {
		err := func() error {
			defer func() {
				if r := recover(); r != nil {
					stack := string(debug.Stack())
					log.Ctx(ctx).Error().
						Str("panic", fmt.Sprintf("[%T] job scheduler panic: %v", r, r)).
						Msg(stack)
				}
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()

			case newTime := <-s.signal:
				dur := timer.RescheduleEarlier(newTime)
				if dur > 0 {
					log.Ctx(ctx).Trace().
						Msgf("job scheduler: update of scheduled job processing time... runs in %s", dur)
				}
				return nil

			case now := <-timer.Ch():
				count, nextExec, gotAllJobs, err := s.processReadyJobs(ctx, now)

				// If the next processing time isn't known use the default.
				if nextExec.IsZero() {
					const period = time.Minute
					nextExec = now.Add(period)
				}

				// Reset the timer. Make the timer edgy if there are more jobs available.
				dur := timer.ResetAt(nextExec, !gotAllJobs)

				if err != nil {
					log.Ctx(ctx).Err(err).
						Msgf("job scheduler: failed to process jobs; next iteration in %s", dur)
				} else {
					log.Ctx(ctx).Trace().
						Msgf("job scheduler: started %d jobs; next iteration in %s", count, dur)
				}

				return nil
			}
		}()
		if err != nil {
			return err
		}
	}
}

// WaitJobsDone waits until execution of all jobs has finished.
// It is intended to be used for graceful shutdown, after the Run method has finished.
func (s *Scheduler) WaitJobsDone(ctx context.Context) {
	log.Ctx(ctx).Debug().Msg("job scheduler: stopping... waiting for the currently running jobs to finish")

	ch := make(chan struct{})
	go func() {
		s.wgRunning.Wait()
		close(ch)
	}()

	select {
	case <-ctx.Done():
		log.Ctx(ctx).Warn().Msg("job scheduler: stop interrupted")
	case <-ch:
		log.Ctx(ctx).Info().Msg("job scheduler: gracefully stopped")
	}
}

func (s *Scheduler) CancelJob(ctx context.Context, jobUID string) error {
	s.cancelJobMx.Lock()
	cancelFn, ok := s.cancelJobMap[jobUID]
	s.cancelJobMx.Unlock()

	if ok {
		cancelFn()
		return nil
	}

	return s.pubsubService.Publish(ctx, PubSubTopicCancelJob, []byte(jobUID))
}

func (s *Scheduler) handleCancelJob(payload []byte) error {
	jobUID := string(payload)
	if jobUID == "" {
		return nil
	}

	s.cancelJobMx.Lock()
	cancelFn, ok := s.cancelJobMap[jobUID]
	s.cancelJobMx.Unlock()

	if ok {
		cancelFn()
	}

	return nil
}

// scheduleProcessing triggers processing of ready jobs.
// This should be run after adding new jobs to the database.
func (s *Scheduler) scheduleProcessing(scheduled time.Time) {
	go func() {
		select {
		case <-s.done:
		case s.signal <- scheduled:
		}
	}()
}

// scheduleIfHaveMoreJobs triggers processing of ready jobs if the timer is edgy.
// The timer would be edgy if the previous iteration found more jobs that it could start (full capacity).
// This should be run after a non-recurring job has finished.
func (s *Scheduler) scheduleIfHaveMoreJobs() {
	s.scheduleProcessing(time.Time{}) // zero time will trigger the timer if it's edgy
}

// RunJob runs a single job of the provided type.
// All parameters a job receives must be inside the data string (as JSON or whatever the job handler can interpret).
// The caller gets the job UID which can then by used to track execution (the job state and progress percentage).
func (s *Scheduler) RunJob(ctx context.Context,
	jobType string,
	maxRetries int,
	timeout time.Duration,
	data string,
) (string, error) {
	if jobType == "" {
		return "", errors.New("jobType must not be empty")
	}

	if maxRetries < 0 {
		return "", errors.New("maxRetries must be positive")
	}

	if timeout < time.Second {
		return "", errors.New("timeout too short")
	}

	uid, err := UID()
	if err != nil {
		return "", fmt.Errorf("failed to generate job UID: %w", err)
	}

	nowMilli := time.Now().UnixMilli()

	job := &types.Job{
		UID:                 uid,
		Created:             nowMilli,
		Updated:             nowMilli,
		Type:                jobType,
		Priority:            enum.JobPriorityNormal,
		Data:                data,
		Result:              "",
		MaxDurationSeconds:  int(timeout / time.Second),
		MaxRetries:          maxRetries,
		State:               enum.JobStateScheduled,
		Scheduled:           nowMilli,
		TotalExecutions:     0,
		RunBy:               "",
		RunDeadline:         nowMilli,
		RunProgress:         ProgressMin,
		LastExecuted:        0, // never executed
		IsRecurring:         false,
		RecurringCron:       "",
		ConsecutiveFailures: 0,
		LastFailureError:    "",
	}

	ctx = log.Ctx(ctx).With().
		Str("job.UID", job.UID).
		Str("job.Type", job.Type).
		Logger().WithContext(ctx)

	mx, err := globalLock(ctx, s.mxManager)
	if err != nil {
		return "", fmt.Errorf("failed to obtain global lock to add new job: %w", err)
	}

	defer func() {
		if err := mx.Unlock(ctx); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to release global lock after adding a new job")
		}
	}()

	err = s.store.Create(ctx, job)
	if err != nil {
		return "", fmt.Errorf("failed to add new job to the database: %w", err)
	}

	available, err := s.availableSlots(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to count available slots for job execution: %w", err)
	}

	if available == 0 {
		// already running at the full capacity: the job is scheduled, we're done.
		return uid, nil
	}

	err = s.runJob(ctx, job)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to run job")
	}

	return uid, nil
}

// processReadyJobs executes jobs that are ready to run. This function is periodically run by the Scheduler.
// The function returns the number of jobs it has is started, the next scheduled execution time (of this function)
// and a bool value if all currently available ready jobs were started.
// Internally the Scheduler uses an "edgy" timer to reschedule calls of this function.
// The edgy option of the timer will be on if this function hasn't been able to start all job that are ready to run.
// If the timer has the edgy option turned on it will trigger the timer (and thus this function will be called)
// when any currently running job finishes successfully or fails.
func (s *Scheduler) processReadyJobs(ctx context.Context, now time.Time) (int, time.Time, bool, error) {
	mx, err := globalLock(ctx, s.mxManager)
	if err != nil {
		return 0, time.Time{}, false,
			fmt.Errorf("failed to obtain global lock to periodically process ready jobs: %w", err)
	}

	defer func() {
		if err := mx.Unlock(ctx); err != nil {
			log.Ctx(ctx).Err(err).
				Msg("failed to release global lock after periodic processing of ready jobs")
		}
	}()

	availableCount, err := s.availableSlots(ctx)
	if err != nil {
		return 0, time.Time{}, false,
			fmt.Errorf("failed to count available slots for job execution: %w", err)
	}

	// get one over the limit to check if all ready jobs are fetched
	jobs, err := s.store.ListReady(ctx, now, availableCount+1)
	if err != nil {
		return 0, time.Time{}, false,
			fmt.Errorf("failed to load scheduled jobs: %w", err)
	}

	var (
		countExecuted     int
		knownNextExecTime time.Time
		gotAllJobs        bool
	)

	if len(jobs) > availableCount {
		// More jobs are ready than we are able to run.
		jobs = jobs[:availableCount]
	} else {
		gotAllJobs = true
		knownNextExecTime, err = s.store.NextScheduledTime(ctx, now)
		if err != nil {
			return 0, time.Time{}, false,
				fmt.Errorf("failed to read next scheduled time: %w", err)
		}
	}

	for _, job := range jobs {
		jobCtx := log.Ctx(ctx).With().
			Str("job.UID", job.UID).
			Str("job.Type", job.Type).
			Logger().WithContext(ctx)

		err = s.runJob(jobCtx, job)
		if err != nil {
			knownNextExecTime = time.Time{}
			gotAllJobs = false
			log.Ctx(jobCtx).Err(err).Msg("failed to run job")
			continue
		}

		countExecuted++
	}

	return countExecuted, knownNextExecTime, gotAllJobs, nil
}

func (s *Scheduler) availableSlots(ctx context.Context) (int, error) {
	countRunning, err := s.store.CountRunning(ctx)
	if err != nil {
		return 0, err
	}

	availableCount := s.maxRunning - countRunning
	if availableCount < 0 {
		return 0, nil
	}

	return availableCount, nil
}

// runJob updates the job in the database and starts it in a separate goroutine.
// The function will also log the execution.
func (s *Scheduler) runJob(ctx context.Context, job *types.Job) error {
	// Update the job fields for the new execution
	s.preExec(job)

	if err := s.store.UpdateExecution(ctx, job); err != nil {
		return err
	}

	// tell everybody that a job has started
	if err := publishStateChange(ctx, s.pubsubService, job); err != nil {
		log.Err(err).Msg("failed to publish job state change")
	}

	s.wgRunning.Add(1)
	go func(jobCtx context.Context, job *types.Job) {
		defer s.wgRunning.Done()

		log.Ctx(jobCtx).Debug().Msg("started job")

		timeStart := time.Now()

		// Run the job
		execResult, execFailure := s.doExec(jobCtx, job)

		// Update the job fields, reschedule if necessary.
		postExec(job, execResult, execFailure)

		// Use the context.Background() because we want to update the job even if the job's context is done.
		// The context can be done because the job exceeded its deadline or the server is shutting down.
		backgroundCtx := context.Background()

		// tell everybody that a job has finished execution
		if err := publishStateChange(backgroundCtx, s.pubsubService, job); err != nil {
			log.Ctx(jobCtx).Err(err).Msg("failed to publish job state change")
		}

		if mx, err := globalLock(backgroundCtx, s.mxManager); err != nil {
			// If locking failed, just log the error and proceed to update the DB anyway.
			log.Ctx(jobCtx).Err(err).Msg("failed to obtain global lock to update job after execution")
		} else {
			defer func() {
				if err := mx.Unlock(backgroundCtx); err != nil {
					log.Ctx(jobCtx).Err(err).Msg("failed to release global lock to update job after execution")
				}
			}()
		}

		if err := s.store.UpdateExecution(backgroundCtx, job); err != nil {
			log.Ctx(jobCtx).Err(err).Msg("failed to update after execution")
			return
		}

		logInfo := log.Ctx(jobCtx).Info().Str("duration", time.Since(timeStart).String())

		if job.IsRecurring {
			logInfo = logInfo.Bool("job.IsRecurring", true)
		}
		if job.Result != "" {
			logInfo = logInfo.Str("job.Result", job.Result)
		}
		if job.LastFailureError != "" {
			logInfo = logInfo.Str("job.Failure", job.LastFailureError)
		}

		switch job.State {
		case enum.JobStateFinished:
			logInfo.Msg("job successfully finished")
			s.scheduleIfHaveMoreJobs()

		case enum.JobStateFailed:
			logInfo.Msg("job failed")
			s.scheduleIfHaveMoreJobs()

		case enum.JobStateScheduled:
			scheduledTime := time.UnixMilli(job.Scheduled)
			logInfo.
				Str("job.Scheduled", scheduledTime.Format(time.RFC3339Nano)).
				Msg("job finished and rescheduled")

			s.scheduleProcessing(scheduledTime)

		case enum.JobStateRunning:
			log.Ctx(jobCtx).Error().Msg("should not happen; job still has state=running after finishing")
		}
	}(ctx, job)

	return nil
}

// preExec updates the provided types.Job before execution.
func (s *Scheduler) preExec(job *types.Job) {
	if job.MaxDurationSeconds < 1 {
		job.MaxDurationSeconds = 1
	}

	now := time.Now()
	nowMilli := now.UnixMilli()

	execDuration := time.Duration(job.MaxDurationSeconds) * time.Second
	execDeadline := now.Add(execDuration)

	job.Updated = nowMilli
	job.LastExecuted = nowMilli
	job.State = enum.JobStateRunning
	job.RunDeadline = execDeadline.UnixMilli()
	job.RunBy = s.instanceID
	job.RunProgress = ProgressMin
	job.TotalExecutions++
	job.Result = ""
	job.LastFailureError = ""
}

// doExec executes the provided types.Job.
func (s *Scheduler) doExec(ctx context.Context, job *types.Job) (execResult, execError string) {
	execDeadline := time.UnixMilli(job.RunDeadline)

	jobCtx, done := context.WithDeadline(ctx, execDeadline)
	defer done()

	s.cancelJobMx.Lock()
	if _, ok := s.cancelJobMap[job.UID]; ok {
		// should not happen: jobs have unique UIDs!
		s.cancelJobMx.Unlock()
		return "", "failed to start: already running"
	}
	s.cancelJobMap[job.UID] = done
	s.cancelJobMx.Unlock()

	defer func() {
		s.cancelJobMx.Lock()
		delete(s.cancelJobMap, job.UID)
		s.cancelJobMx.Unlock()
	}()

	execResult, err := s.executor.exec(jobCtx, job.UID, job.Type, job.Data)
	if err != nil {
		execError = err.Error()
	}

	return
}

// postExec updates the provided types.Job after execution and reschedules it if necessary.
func postExec(job *types.Job, resultData, resultErr string) {
	now := time.Now()
	nowMilli := now.UnixMilli()

	job.Updated = nowMilli
	job.Result = resultData
	job.RunBy = ""

	if resultErr != "" {
		job.ConsecutiveFailures++
		job.State = enum.JobStateFailed
		job.LastFailureError = resultErr
	} else {
		job.State = enum.JobStateFinished
		job.RunProgress = ProgressMax
	}

	// Reschedule recurring jobs
	if job.IsRecurring {
		if resultErr == "" {
			job.ConsecutiveFailures = 0
		}

		exp, err := cronexpr.Parse(job.RecurringCron)
		if err != nil {
			job.State = enum.JobStateFailed

			messages := fmt.Sprintf("failed to parse cron string: %s", err.Error())
			if job.LastFailureError != "" {
				messages = messages + "; " + job.LastFailureError
			}

			job.LastFailureError = messages
		} else {
			job.State = enum.JobStateScheduled
			job.Scheduled = exp.Next(now).UnixMilli()
		}

		return
	}

	// Reschedule the failed job if retrying is allowed
	if job.State == enum.JobStateFailed && job.ConsecutiveFailures <= job.MaxRetries {
		const retryDelay = 15 * time.Second
		job.State = enum.JobStateScheduled
		job.Scheduled = now.Add(retryDelay).UnixMilli()
		job.RunProgress = ProgressMin
	}
}

func (s *Scheduler) AddRecurring(
	ctx context.Context,
	jobUID,
	jobType,
	cronDef string,
	maxDur time.Duration,
) error {
	cronExp, err := cronexpr.Parse(cronDef)
	if err != nil {
		return fmt.Errorf("invalid cron definition string for job type=%s: %w", jobType, err)
	}

	now := time.Now()
	nowMilli := now.UnixMilli()

	nextExec := cronExp.Next(now)

	job := &types.Job{
		UID:                 jobUID,
		Created:             nowMilli,
		Updated:             nowMilli,
		Type:                jobType,
		Priority:            enum.JobPriorityElevated,
		Data:                "",
		Result:              "",
		MaxDurationSeconds:  int(maxDur / time.Second),
		MaxRetries:          0,
		State:               enum.JobStateScheduled,
		Scheduled:           nextExec.UnixMilli(),
		TotalExecutions:     0,
		RunBy:               "",
		RunDeadline:         0,
		RunProgress:         0,
		LastExecuted:        0,
		IsRecurring:         true,
		RecurringCron:       cronDef,
		ConsecutiveFailures: 0,
		LastFailureError:    "",
	}

	err = s.store.Upsert(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to upsert job id=%s type=%s: %w", jobUID, jobType, err)
	}

	return nil
}

func (s *Scheduler) createNecessaryJobs(ctx context.Context) error {
	mx, err := globalLock(ctx, s.mxManager)
	if err != nil {
		return fmt.Errorf("failed to obtain global lock to create necessary jobs: %w", err)
	}

	defer func() {
		if err := mx.Unlock(ctx); err != nil {
			log.Ctx(ctx).Err(err).
				Msg("failed to release global lock after creating necessary jobs")
		}
	}()

	err = s.AddRecurring(ctx, jobUIDPurge, jobTypePurge, jobCronPurge, 5*time.Second)
	if err != nil {
		return err
	}

	err = s.AddRecurring(ctx, jobUIDOverdue, jobTypeOverdue, jobCronOverdue, 5*time.Second)
	if err != nil {
		return err
	}

	return nil
}

// registerNecessaryJobs registers two jobs: overdue job recovery and purge old finished jobs.
// These two jobs types are integral part of the job scheduler.
func (s *Scheduler) registerNecessaryJobs() error {
	handlerOverdue := newJobOverdue(s.store, s.mxManager, s)
	err := s.executor.Register(jobTypeOverdue, handlerOverdue)
	if err != nil {
		return err
	}

	handlerPurge := newJobPurge(s.store, s.mxManager, s.purgeMinOldAge)
	err = s.executor.Register(jobTypePurge, handlerPurge)
	if err != nil {
		return err
	}

	return nil
}
