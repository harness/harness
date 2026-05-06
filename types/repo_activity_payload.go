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
	"fmt"

	"github.com/harness/gitness/types/enum"
)

// RepoActivityPayload identifies typed payloads for repository activities.
// The approach mirrors pull request activity payload typing.
type RepoActivityPayload interface {
	ActivityType() enum.RepoActivityType
}

// RepoActivityPayloadBranchCreated is payload for branch creation activities.
type RepoActivityPayloadBranchCreated struct {
	Name string `json:"name"`
	New  string `json:"new"`
}

func (RepoActivityPayloadBranchCreated) ActivityType() enum.RepoActivityType {
	return enum.RepoActivityTypeBranchCreated
}

// RepoActivityPayloadBranchUpdated is payload for branch update activities.
type RepoActivityPayloadBranchUpdated struct {
	Name   string `json:"name"`
	Old    string `json:"old"`
	New    string `json:"new"`
	Forced bool   `json:"forced,omitempty"`
}

func (RepoActivityPayloadBranchUpdated) ActivityType() enum.RepoActivityType {
	return enum.RepoActivityTypeBranchUpdated
}

// RepoActivityPayloadBranchDeleted is payload for branch deletion activities.
type RepoActivityPayloadBranchDeleted struct {
	Name string `json:"name"`
	Old  string `json:"old"`
}

func (RepoActivityPayloadBranchDeleted) ActivityType() enum.RepoActivityType {
	return enum.RepoActivityTypeBranchDeleted
}

// repoActivityPayloadFactoryMethod is an alias for a function that creates a new RepoActivityPayload.
type repoActivityPayloadFactoryMethod func() RepoActivityPayload

// allRepoActivityPayloads maps activity types to payload factory methods.
var allRepoActivityPayloads = func(
	factoryMethods []repoActivityPayloadFactoryMethod,
) map[enum.RepoActivityType]repoActivityPayloadFactoryMethod {
	payloadMap := make(map[enum.RepoActivityType]repoActivityPayloadFactoryMethod)
	for _, factoryMethod := range factoryMethods {
		payloadMap[factoryMethod().ActivityType()] = factoryMethod
	}
	return payloadMap
}([]repoActivityPayloadFactoryMethod{
	func() RepoActivityPayload { return &RepoActivityPayloadBranchCreated{} },
	func() RepoActivityPayload { return &RepoActivityPayloadBranchUpdated{} },
	func() RepoActivityPayload { return &RepoActivityPayloadBranchDeleted{} },
})

func newRepoActivityPayload(activityType enum.RepoActivityType) (RepoActivityPayload, error) {
	payloadFactoryMethod, ok := allRepoActivityPayloads[activityType]
	if !ok {
		return nil, fmt.Errorf("repo activity type %q doesn't have a payload", activityType)
	}

	return payloadFactoryMethod(), nil
}

// MarshalRepoActivityPayload marshals a typed payload to JSON.
func MarshalRepoActivityPayload(payload RepoActivityPayload) (json.RawMessage, error) {
	if payload == nil {
		return json.RawMessage("{}"), nil
	}

	res, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal repository activity payload: %w", err)
	}

	return json.RawMessage(res), nil
}

// UnmarshalRepoActivityPayload unmarshals payload JSON based on activity type.
func UnmarshalRepoActivityPayload(
	activityType enum.RepoActivityType,
	raw json.RawMessage,
) (RepoActivityPayload, error) {
	if len(raw) == 0 {
		return newRepoActivityPayload(activityType)
	}

	payload, err := newRepoActivityPayload(activityType)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(raw, payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository activity payload: %w", err)
	}

	return payload, nil
}
