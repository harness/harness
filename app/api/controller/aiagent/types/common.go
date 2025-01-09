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

type Suggestion struct {
	ID             string
	Prompt         string
	UserSuggestion string
	Suggestion     string
}

// Enum type for Role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Enum type for Message Type.
type MessageType string

const (
	TypeText MessageType = "text"
	TypeYAML MessageType = "yaml"
)

type Conversation struct {
	Role    Role    `json:"role"`
	Message Message `json:"message"`
}

type Message struct {
	Type MessageType `json:"type"`
	Data string      `json:"data"`
}

// Additional common structs can be defined here as needed.
