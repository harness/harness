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

package types

import (
	"encoding/json"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusSuccess    TaskStatus = "success"
	TaskStatusFailure    TaskStatus = "failure"
)

type TaskKind string

const (
	TaskKindBuildRegistryIndex   TaskKind = "build_registry_index"
	TaskKindBuildPackageIndex    TaskKind = "build_package_index"
	TaskKindBuildPackageMetadata TaskKind = "build_package_metadata"
)

type SourceType string

const (
	SourceTypeRegistry SourceType = "Registry"
	SourceTypeArtifact SourceType = "Artifact"
)

type Task struct {
	Key       string
	Kind      TaskKind
	Payload   json.RawMessage
	Status    TaskStatus
	RunAgain  bool
	UpdatedAt time.Time
}

type TaskSource struct {
	Key       string
	SrcType   SourceType
	SrcID     int64
	Status    TaskStatus
	RunID     *string
	Error     *string
	UpdatedAt time.Time
}

type TaskEvent struct {
	ID      string
	Key     string
	Event   string
	Payload *json.RawMessage
	At      time.Time
}

type SourceRef struct {
	Type SourceType `json:"type"`
	ID   int64      `json:"id"`
}

type BuildRegistryIndexTaskPayload struct {
	Key         string `json:"key"`
	RegistryID  int64  `json:"registry_id"`
	PrincipalID int64  `json:"principal_id"` // TODO: setting service principal ID to run the task
}

type BuildPackageIndexTaskPayload struct {
	Key         string `json:"key"`
	RegistryID  int64  `json:"registry_id"`
	Image       string `json:"image"`
	PrincipalID int64  `json:"principal_id"` // TODO: setting service principal ID to run the task
}

type BuildPackageMetadataTaskPayload struct {
	Key         string `json:"key"`
	RegistryID  int64  `json:"registry_id"`
	Image       string `json:"image"`
	Version     string `json:"version"`
	PrincipalID int64  `json:"principal_id"`
}
