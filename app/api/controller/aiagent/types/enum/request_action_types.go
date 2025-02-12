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

package enum

import (
	"github.com/harness/gitness/types/enum"
)

type RequestAction string

const (
	CreateStep     RequestAction = "CREATE_STEP"
	UpdateStep     RequestAction = "UPDATE_STEP"
	CreateStage    RequestAction = "CREATE_STAGE"
	UpdateStage    RequestAction = "UPDATE_STAGE"
	CreatePipeline RequestAction = "CREATE_PIPELINE"
	UpdatePipeline RequestAction = "UPDATE_PIPELINE"
)

func (a RequestAction) IsValid() bool {
	validTypes, _ := GetAllRequestActions()
	for _, validType := range validTypes {
		if a == validType {
			return true
		}
	}
	return false
}

func (a RequestAction) Sanitize() (RequestAction, bool) {
	return enum.Sanitize(a, GetAllRequestActions)
}

func GetAllRequestActions() ([]RequestAction, RequestAction) {
	return RequestActions, ""
}

var RequestActions = ([]RequestAction{
	CreatePipeline,
	CreateStage,
	CreateStep,
	UpdatePipeline,
	UpdateStage,
	UpdateStep,
})
