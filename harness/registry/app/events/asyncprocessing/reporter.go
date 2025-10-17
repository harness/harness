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
	"errors"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

const RegistryAsyncProcessing = "registry-async-postprocessing"

// Reporter is the event reporter for this package.
// It exposes typesafe send methods for all events of this package.
// NOTE: Event send methods are in the event's dedicated file.
type Reporter struct {
	tx                   dbtx.Transactor
	innerReporter        *events.GenericReporter
	TaskRepository       store.TaskRepository
	TaskSourceRepository store.TaskSourceRepository
	TaskEventRepository  store.TaskEventRepository
}

func NewReporter(
	tx dbtx.Transactor,
	eventsSystem *events.System,
	taskRepository store.TaskRepository,
	taskSourceRepository store.TaskSourceRepository,
	taskEventRepository store.TaskEventRepository,
) (*Reporter, error) {
	innerReporter, err := events.NewReporter(eventsSystem, RegistryAsyncProcessing)
	if err != nil {
		return nil, errors.New("failed to create new GenericReporter for registry async processing from event system")
	}

	return &Reporter{
		tx:                   tx,
		innerReporter:        innerReporter,
		TaskRepository:       taskRepository,
		TaskSourceRepository: taskSourceRepository,
		TaskEventRepository:  taskEventRepository,
	}, nil
}
