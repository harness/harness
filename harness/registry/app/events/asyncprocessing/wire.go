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
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

func ProvideReaderFactory(eventsSystem *events.System) (*events.ReaderFactory[*Reader], error) {
	return NewReaderFactory(eventsSystem)
}

func ProvideAsyncProcessingReporter(
	tx dbtx.Transactor,
	eventsSystem *events.System,
	taskRepository store.TaskRepository,
	taskSourceRepository store.TaskSourceRepository,
	taskEventRepository store.TaskEventRepository,
) (*Reporter, error) {
	reporter, err := NewReporter(tx, eventsSystem, taskRepository, taskSourceRepository, taskEventRepository)
	if err != nil {
		return nil, err
	}
	return reporter, nil
}
