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
	"github.com/harness/gitness/app/api/controller/aiagent/types/enum"

	"github.com/google/uuid"
)

// Chat represents a chat conversation with a unique ID, prompt, metadata, and conversation history.
type Chat struct {
	ConversationID uuid.UUID         `json:"conversation_id,omitempty"`
	Prompt         string            `json:"prompt" required:"true"`
	Metadata       map[string]string `json:"metadata" required:"true"`
	Conversation   []Conversation    `json:"conversation"`
}

// ChatOutput represents the output of a chat request, including the conversation ID, explanation, and response.
type ChatOutput struct {
	ConversationID uuid.UUID      `json:"conversation_id" required:"true"`
	Explanation    string         `json:"explanation,omitempty"`
	Response       []ChatResponse `json:"response" required:"true"`
}

// ChatResponse represents a response in a chat conversation, including the role and message.
type ChatResponse struct {
	Role    enum.Role       `json:"role" required:"true"`
	Message ResponseMessage `json:"message" required:"true"`
}

// ResponseMessage represents a message in a chat response, including the type, data, and actions.
type ResponseMessage struct {
	Type    enum.MessageType `json:"type" required:"true"`
	Data    string           `json:"data" required:"true"`
	Actions []enum.Action    `json:"actions,omitempty"`
}
