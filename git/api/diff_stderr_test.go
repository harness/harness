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

package api

import (
	"bytes"
	"testing"

	"github.com/harness/gitness/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiffStderr(t *testing.T) {
	tests := []struct {
		name            string
		stderr          string
		wantNil         bool
		wantInvalidArgs bool
	}{
		{name: "empty", stderr: "", wantNil: true},
		{
			name:            "invalid revision range",
			stderr:          "fatal: Invalid revision range abc..def\n",
			wantInvalidArgs: true,
		},
		{
			name:            "invalid symmetric difference",
			stderr:          "fatal: Invalid symmetric difference expression abc...def\n",
			wantInvalidArgs: true,
		},
		{name: "other failure stays unclassified", stderr: "fatal: something else broke\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseDiffStderr(bytes.NewBufferString(tt.stderr))
			if tt.wantNil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantInvalidArgs, errors.IsInvalidArgument(err))
		})
	}
}
