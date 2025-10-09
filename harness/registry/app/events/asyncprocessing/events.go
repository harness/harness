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
	"fmt"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

const ExecuteAsyncTask events.EventType = "execute_async_task"

type ExecuteAsyncTaskPayload struct {
	TaskKey string `json:"task_key"`
}

func (r *Reporter) BuildRegistryIndex(ctx context.Context, registryID int64, sources []types.SourceRef) {
	session, _ := request.AuthSessionFrom(ctx)
	principalID := session.Principal.ID
	r.BuildRegistryIndexWithPrincipal(ctx, registryID, sources, principalID)
}

func (r *Reporter) BuildRegistryIndexWithPrincipal(
	ctx context.Context,
	registryID int64,
	sources []types.SourceRef,
	principalID int64,
) {
	key := fmt.Sprintf("registry_%d", registryID)
	payload, err := json.Marshal(&types.BuildRegistryIndexTaskPayload{
		Key:         key,
		RegistryID:  registryID,
		PrincipalID: principalID,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
	}
	task := &types.Task{
		Key:     key,
		Kind:    types.TaskKindBuildRegistryIndex,
		Payload: payload,
	}
	sources = append(sources, types.SourceRef{Type: types.SourceTypeRegistry, ID: registryID})
	err = r.upsertAndSendEvent(ctx, task, sources)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
	}
}

func (r *Reporter) BuildPackageIndex(ctx context.Context, registryID int64, image string) {
	session, _ := request.AuthSessionFrom(ctx)
	principalID := session.Principal.ID
	r.BuildPackageIndexWithPrincipal(ctx, registryID, image, principalID)
}

func (r *Reporter) BuildPackageIndexWithPrincipal(
	ctx context.Context, registryID int64, image string, principalID int64,
) {
	key := fmt.Sprintf("package_%d_%s", registryID, image)
	payload, err := json.Marshal(&types.BuildPackageIndexTaskPayload{
		Key:         key,
		RegistryID:  registryID,
		Image:       image,
		PrincipalID: principalID,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
	}
	task := &types.Task{
		Key:     key,
		Kind:    types.TaskKindBuildPackageIndex,
		Payload: payload,
	}

	sources := make([]types.SourceRef, 0)
	sources = append(sources, types.SourceRef{Type: types.SourceTypeRegistry, ID: registryID})
	err = r.upsertAndSendEvent(ctx, task, sources)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
	}
}

func (r *Reporter) BuildPackageMetadata(
	ctx context.Context, registryID int64, image string, version string,
) {
	session, _ := request.AuthSessionFrom(ctx)
	principalID := session.Principal.ID
	r.BuildPackageMetadataWithPrincipal(ctx, registryID, image, version, principalID)
}

func (r *Reporter) BuildPackageMetadataWithPrincipal(
	ctx context.Context, registryID int64, image string,
	version string, principalID int64,
) {
	key := fmt.Sprintf("package_%d_%s_%s_metadata", registryID, image, version)
	payload, err := json.Marshal(&types.BuildPackageMetadataTaskPayload{
		Key:         key,
		RegistryID:  registryID,
		Image:       image,
		Version:     version,
		PrincipalID: principalID,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
	}
	task := &types.Task{
		Key:     key,
		Kind:    types.TaskKindBuildPackageMetadata,
		Payload: payload,
	}

	sources := make([]types.SourceRef, 0)
	sources = append(sources, types.SourceRef{Type: types.SourceTypeRegistry, ID: registryID})
	err = r.upsertAndSendEvent(ctx, task, sources)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
	}
}

func (r *Reporter) upsertAndSendEvent(
	ctx context.Context,
	task *types.Task,
	sources []types.SourceRef,
) error {
	shouldEnqueue, err := r.upsertTask(ctx, task, sources)

	if err == nil && shouldEnqueue {
		payload := &ExecuteAsyncTaskPayload{
			TaskKey: task.Key,
		}
		eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ExecuteAsyncTask, payload)
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to send execute async task event")
			return err
		}
		log.Ctx(ctx).Debug().Msgf("reported execute async task event with id '%s'", eventID)
	}
	return err
}

//nolint:nestif
func (r *Reporter) upsertTask(ctx context.Context, task *types.Task, sources []types.SourceRef) (bool, error) {
	shouldEnqueue := false
	err := r.tx.WithTx(
		ctx, func(ctx context.Context) error {
			err := r.TaskRepository.UpsertTask(ctx, task)
			if err != nil {
				return fmt.Errorf("failed to upsert task: %w", err)
			}
			status, err := r.TaskRepository.LockForUpdate(ctx, task)
			if err != nil {
				return fmt.Errorf("failed to lock task %s for update: %w", task.Key, err)
			}

			for _, src := range sources {
				err = r.TaskSourceRepository.InsertSource(ctx, task.Key, src)
				if err != nil {
					return fmt.Errorf("failed to insert source %s for task %s: %w", src.Type, task.Key, err)
				}
			}

			if status == types.TaskStatusProcessing {
				err = r.TaskRepository.SetRunAgain(ctx, task.Key, true)
				if err != nil {
					return fmt.Errorf("failed to set task %s to run again: %w", task.Key, err)
				}
				err = r.TaskEventRepository.LogTaskEvent(ctx, task.Key, "merged", task.Payload)
				if err != nil {
					log.Ctx(ctx).Error().Msgf("failed to log task event for task %s: %v", task.Key, err)
				}
			} else {
				err = r.TaskRepository.UpdateStatus(ctx, task.Key, types.TaskStatusPending)
				if err != nil {
					return fmt.Errorf("failed to update task %s status to pending: %w", task.Key, err)
				}
				err = r.TaskEventRepository.LogTaskEvent(ctx, task.Key, "enqueued", task.Payload)
				if err != nil {
					return fmt.Errorf("failed to log task event for task %s: %w", task.Key, err)
				}
				shouldEnqueue = true
			}
			return nil
		})
	return shouldEnqueue, err
}

func (r *Reader) RegisterExecuteAsyncTask(
	fn events.HandlerFunc[*ExecuteAsyncTaskPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ExecuteAsyncTask, fn, opts...)
}
