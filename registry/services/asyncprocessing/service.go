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

package asyncprocessing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/utils/cargo"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/stream"

	"github.com/rs/zerolog/log"
)

const (
	timeout               = 1 * time.Hour
	eventsReaderGroupName = "registry:postprocessing"
)

type Service struct {
	tx                     dbtx.Transactor
	rpmRegistryHelper      RpmHelper
	cargoRegistryHelper    cargo.RegistryHelper
	locker                 *locker.Locker
	registryDao            store.RegistryRepository
	taskRepository         store.TaskRepository
	taskSourceRepository   store.TaskSourceRepository
	taskEventRepository    store.TaskEventRepository
	innerReporter          *events.GenericReporter
	postProcessingReporter *asyncprocessing.Reporter
}

func NewService(
	ctx context.Context,
	tx dbtx.Transactor,
	rpmRegistryHelper RpmHelper,
	cargoRegistryHelper cargo.RegistryHelper,
	locker *locker.Locker,
	artifactsReaderFactory *events.ReaderFactory[*asyncprocessing.Reader],
	config Config,
	registryDao store.RegistryRepository,
	taskRepository store.TaskRepository,
	taskSourceRepository store.TaskSourceRepository,
	taskEventRepository store.TaskEventRepository,
	eventsSystem *events.System,
	postProcessingReporter *asyncprocessing.Reporter,
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided postprocessing service config is invalid: %w", err)
	}
	innerReporter, err := events.NewReporter(eventsSystem, asyncprocessing.RegistryAsyncProcessing)
	if err != nil {
		return nil, errors.New("failed to create new GenericReporter for registry async processing from event system")
	}
	s := &Service{
		rpmRegistryHelper:      rpmRegistryHelper,
		cargoRegistryHelper:    cargoRegistryHelper,
		locker:                 locker,
		tx:                     tx,
		registryDao:            registryDao,
		taskRepository:         taskRepository,
		taskSourceRepository:   taskSourceRepository,
		taskEventRepository:    taskEventRepository,
		innerReporter:          innerReporter,
		postProcessingReporter: postProcessingReporter,
	}
	_, err = artifactsReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *asyncprocessing.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			// register events with common wrapper
			_ = r.RegisterExecuteAsyncTask(wrapHandler(
				s.handleEventExecuteAsyncTask,
			))

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch registry event reader for postprocessing: %w", err)
	}
	return s, nil
}

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
	AllowLoopback   bool
}

func (c *Config) Prepare() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.EventReaderName == "" {
		return errors.New("Config.EventReaderName is required")
	}
	if c.Concurrency < 1 {
		return errors.New("Config.Concurrency has to be a positive number")
	}
	if c.MaxRetries < 0 {
		return errors.New("Config.MaxRetries can't be negative")
	}
	return nil
}

func wrapHandler[T any](
	handler events.HandlerFunc[T],
) events.HandlerFunc[T] {
	return func(ctx context.Context, e *events.Event[T]) error {
		return handler(ctx, e)
	}
}

func (s *Service) handleEventExecuteAsyncTask(
	ctx context.Context,
	e *events.Event[*asyncprocessing.ExecuteAsyncTaskPayload],
) error {
	unlock, err := s.locker.LockResource(ctx, e.Payload.TaskKey, timeout)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed acquire lock by key, eventID: %s, eventId:%s err: %v", e.Payload.TaskKey, e.ID, err)
		return err
	}
	defer unlock()

	task, err := s.taskRepository.Find(ctx, e.Payload.TaskKey)
	if err != nil {
		return err
	}

	err = s.ProcessingStatusUpdate(ctx, task, e.ID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	var processingErr error
	//nolint:nestif
	if task.Kind == types.TaskKindBuildRegistryIndex {
		var payload types.BuildRegistryIndexTaskPayload
		err = json.Unmarshal(task.Payload, &payload)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to unmarshal task payload for task [%s]: %v", task.Key, err)
			return fmt.Errorf("failed to unmarshal task payload: %w", err)
		}
		registry, err := s.registryDao.Get(ctx, payload.RegistryID)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to get registry [%d] for registry build index event: %s, err: %v",
				payload.RegistryID, e.ID, err)
			return fmt.Errorf("failed to get registry: %w", err)
		}
		//nolint:exhaustive
		switch registry.PackageType {
		case artifact.PackageTypeRPM:
			err := s.rpmRegistryHelper.BuildRegistryFiles(ctx, *registry, payload.PrincipalID)
			if err != nil {
				processingErr = fmt.Errorf("failed to build RPM registry files for registry [%d]: %w",
					payload.RegistryID, err)
			}
			if registry.Type != artifact.RegistryTypeVIRTUAL {
				registryIDs, err2 := s.registryDao.FetchRegistriesIDByUpstreamProxyID(
					ctx, strconv.FormatInt(registry.ID, 10), registry.RootParentID,
				)
				if err2 != nil {
					log.Ctx(ctx).Error().Msgf("failed to fetch registries whyle building registry "+
						"files by upstream proxy ID for registry [%d]: %v", payload.RegistryID, err2)
				}
				if len(registryIDs) > 0 {
					for _, id := range registryIDs {
						s.postProcessingReporter.BuildRegistryIndexWithPrincipal(
							ctx, id, make([]types.SourceRef, 0), payload.PrincipalID,
						)
					}
				}
			}

		default:
			log.Ctx(ctx).Error().Msgf("unsupported package type [%s] for registry [%d] in task [%s]",
				registry.PackageType, payload.RegistryID, task.Key)
		}
	} else if task.Kind == types.TaskKindBuildPackageIndex {
		var payload types.BuildPackageIndexTaskPayload
		err = json.Unmarshal(task.Payload, &payload)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to unmarshal task payload for task [%s]: %v", task.Key, err)
			return fmt.Errorf("failed to unmarshal task payload: %w", err)
		}
		registry, err := s.registryDao.Get(ctx, payload.RegistryID)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to get registry [%d] for registry build index event: %s, err: %v",
				payload.RegistryID, e.ID, err)
			return fmt.Errorf("failed to get registry: %w", err)
		}
		//nolint:exhaustive
		switch registry.PackageType {
		case artifact.PackageTypeCARGO:
			err := s.cargoRegistryHelper.UpdatePackageIndex(
				ctx, payload.PrincipalID, registry.RootParentID, registry.ID, payload.Image,
			)
			if err != nil {
				processingErr = fmt.Errorf("failed to build CARGO package index for registry [%d] package [%s]: %w",
					payload.RegistryID, payload.Image, err)
			}
		default:
			log.Ctx(ctx).Error().Msgf("unsupported package type [%s] for registry [%d] and image [%s] in task [%s]",
				registry.PackageType, payload.RegistryID, payload.Image, task.Key)
		}
	}

	runAgain, err := s.finalStatusUpdate(ctx, e, task, processingErr)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to update final status for task [%s]: %v", task.Key, err)
	}

	if runAgain {
		eventID, err := events.ReporterSendEvent(s.innerReporter, ctx, asyncprocessing.ExecuteAsyncTask, e.Payload)
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
			return err
		}
		log.Ctx(ctx).Debug().Msgf("reported execute async task event with id '%s'", eventID)
	}
	return nil
}

//nolint:nestif
func (s *Service) finalStatusUpdate(
	ctx context.Context,
	e *events.Event[*asyncprocessing.ExecuteAsyncTaskPayload],
	task *types.Task,
	processingErr error,
) (bool, error) {
	var runAgain bool
	err := s.tx.WithTx(
		ctx, func(ctx context.Context) error {
			_, err := s.taskRepository.LockForUpdate(ctx, task)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("failed to lock task [%s] for update: %v", task.Key, err)
				return fmt.Errorf("failed to lock task for update: %w", err)
			}
			if processingErr != nil {
				log.Error().Ctx(ctx).Msgf("processing error for task [%s]: %v", task.Key, processingErr)
				err = s.taskSourceRepository.UpdateSourceStatus(ctx, e.ID, types.TaskStatusFailure, processingErr.Error())
				if err != nil {
					return err
				}
				runAgain, err = s.taskRepository.CompleteTask(ctx, task.Key, types.TaskStatusFailure)
				if err != nil {
					return err
				}
			} else {
				err = s.taskSourceRepository.UpdateSourceStatus(ctx, e.ID, types.TaskStatusSuccess, "")
				if err != nil {
					return err
				}
				runAgain, err = s.taskRepository.CompleteTask(ctx, task.Key, types.TaskStatusSuccess)
				if err != nil {
					return err
				}
			}
			return err
		})
	if err != nil {
		return false, fmt.Errorf("failed to update final statuses of task and sources, eventID:%s, task key: %s, err: %w",
			e.ID, task.Key, err)
	}
	return runAgain, nil
}

func (s *Service) ProcessingStatusUpdate(ctx context.Context, task *types.Task, runID string) error {
	err := s.tx.WithTx(
		ctx, func(ctx context.Context) error {
			_, err := s.taskRepository.LockForUpdate(ctx, task)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("failed to lock task [%s] for update: %v", task.Key, err)
				return fmt.Errorf("failed to lock task for update: %w", err)
			}
			err = s.taskRepository.UpdateStatus(ctx, task.Key, types.TaskStatusProcessing)
			if err != nil {
				return err
			}
			err = s.taskRepository.SetRunAgain(ctx, task.Key, false)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("failed to set task [%s] to run again: %v", task.Key, err)
				return fmt.Errorf("failed to set task to run again: %w", err)
			}
			err = s.taskSourceRepository.ClaimSources(ctx, task.Key, runID)
			if err != nil {
				return err
			}
			err = s.taskEventRepository.LogTaskEvent(ctx, task.Key, "started", task.Payload)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("failed to log task event for task [%s]: %v", task.Key, err)
			}
			return nil
		},
	)
	return err
}
