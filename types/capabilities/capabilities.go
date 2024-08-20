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

package capabilities

import "context"

type AIContextPayload interface {
	GetType() AIContextPayloadType
	GetName() string
}

type AIContextPayloadType string
type AIContext struct {
	Type    AIContextPayloadType `json:"type"`
	Payload AIContextPayload     `json:"payload"`
	Name    string               `json:"name"`
}

type CapabilityReference struct {
	Type    Type    `json:"type"`
	Version Version `json:"version"`
}

type Type string
type Version string

type Capabilities struct {
	Capabilities []Capability `json:"capabilities"`
}

type Capability struct {
	Type         Type         `json:"type"`
	NewInput     func() Input `json:"-"`
	Logic        Logic        `json:"-"`
	Version      Version      `json:"version"`
	ReturnToUser bool         `json:"return_to_user"`
}

type Logic func(ctx context.Context, input Input) (Output, error)

type Input interface {
	IsCapabilityInput()
}

type Output interface {
	IsCapabilityOutput()
	GetType() AIContextPayloadType
	GetName() string
}
