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

package aiagent

import (
	"testing"

	"github.com/harness/gitness/app/api/controller/aiagent/types"
	"github.com/harness/gitness/app/api/controller/aiagent/types/enum"
	"github.com/harness/gitness/app/api/usererror"

	"github.com/stretchr/testify/assert"
)

type mockSanitizableInput struct {
	conversations []types.Conversation
}

func (m *mockSanitizableInput) GetConversation() []types.Conversation {
	return m.conversations
}

func TestSanitizeConversation(t *testing.T) {
	rolesStr := toString(enum.GetAllRoles)
	messageTypeStr := toString(enum.GetAllMessageTypes)

	tests := []struct {
		name  string
		input *mockSanitizableInput
		want  error
	}{
		{
			name: "valid conversation",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{
					{
						Role: enum.RoleUser,
						Message: &types.Message{
							Type: enum.TypeText,
							Data: "data",
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "missing role",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{
					{
						Role: "",
						Message: &types.Message{
							Type: enum.TypeText,
							Data: "data",
						},
					},
				},
			},
			want: usererror.BadRequest("role must be provided"),
		},
		{
			name: "invalid role",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{
					{
						Role: "invalidRole",
						Message: &types.Message{
							Type: enum.TypeText,
							Data: "data",
						},
					},
				},
			},
			want: usererror.BadRequestf(
				"invalid type given for Role: invalidRole, allowed values are '%s'", rolesStr),
		},
		{
			name: "missing message type",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{
					{
						Role: enum.RoleUser,
						Message: &types.Message{
							Type: "",
							Data: "data",
						},
					},
				},
			},
			want: usererror.BadRequestf("message type must be provided"),
		},
		{
			name: "invalid message type",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{
					{
						Role: enum.RoleUser,
						Message: &types.Message{
							Type: "invalidMessageType",
							Data: "data",
						},
					},
				},
			},
			want: usererror.BadRequestf(
				"invalid type given for message type: invalidMessageType, allowed values are %s", messageTypeStr),
		},
		{
			name: "missing message data",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{
					{
						Role: enum.RoleUser,
						Message: &types.Message{
							Type: enum.TypeText,
							Data: "",
						},
					},
				},
			},
			want: usererror.BadRequest("message data must be provided"),
		},
		{
			name: "empty conversation",
			input: &mockSanitizableInput{
				conversations: []types.Conversation{},
			},
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SanitizeConversation(test.input)
			if test.want != nil {
				assert.EqualError(t, got, test.want.Error())
			} else {
				assert.NoError(t, got)
			}
		})
	}
}
