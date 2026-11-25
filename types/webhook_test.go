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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookMarshalJSON_RedactsMaskedHeaders(t *testing.T) {
	hook := &Webhook{
		Identifier: "hook",
		Secret:     "encrypted-secret",
		ExtraHeaders: []ExtraHeader{
			{Key: "Authorization", Value: "Bearer super-secret-token", Masked: true},
			{Key: "X-Team", Value: "platform"},
		},
	}

	raw, err := json.Marshal(hook)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(raw), "super-secret-token"), "masked value leaked: %s", raw)
	assert.False(t, strings.Contains(string(raw), "encrypted-secret"), "secret leaked: %s", raw)

	var out struct {
		HasSecret    bool                         `json:"has_secret"`
		ExtraHeaders []map[string]json.RawMessage `json:"extra_headers"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	assert.True(t, out.HasSecret)
	require.Len(t, out.ExtraHeaders, 2)

	_, hasValue := out.ExtraHeaders[0]["value"]
	assert.False(t, hasValue, "masked header must not contain a value field")
	assert.JSONEq(t, `"Authorization"`, string(out.ExtraHeaders[0]["key"]))
	assert.JSONEq(t, `true`, string(out.ExtraHeaders[0]["masked"]))

	assert.JSONEq(t, `"platform"`, string(out.ExtraHeaders[1]["value"]))

	// the original object must not be modified (it's still needed to deliver the webhook)
	assert.Equal(t, "Bearer super-secret-token", hook.ExtraHeaders[0].Value)
}

func TestWebhookMarshalJSON_List(t *testing.T) {
	// list APIs return slices of webhooks, which must be redacted as well.
	hooks := []*Webhook{{ExtraHeaders: []ExtraHeader{{Key: "K", Value: "secret-value", Masked: true}}}}

	raw, err := json.Marshal(hooks)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(raw), "secret-value"), "masked value leaked: %s", raw)
}

func TestWebhookMarshalJSON_NoHeaders(t *testing.T) {
	raw, err := json.Marshal(&Webhook{Identifier: "hook"})
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(raw), "extra_headers"))
}

func TestRedactExtraHeaders(t *testing.T) {
	assert.Nil(t, RedactExtraHeaders(nil))

	in := []ExtraHeader{{Key: "A", Value: "a", Masked: true}, {Key: "B", Value: "b"}}
	out := RedactExtraHeaders(in)
	assert.Equal(t, []ExtraHeader{{Key: "A", Masked: true}, {Key: "B", Value: "b"}}, out)
	assert.Equal(t, "a", in[0].Value, "input must not be modified")
}
