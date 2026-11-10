// Copyright 2026 Harness, Inc.
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
	"testing"

	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCodeOwnersValidation_AddAtLine(t *testing.T) {
	var v CodeOwnersValidation
	v.AddAtLine(enum.CodeOwnerViolationCodePatternEmpty, 7, "empty pattern")

	require.Len(t, v.Violations, 1)
	require.Equal(t, enum.CodeOwnerViolationCodePatternEmpty, v.Violations[0].Code)
	require.Equal(t, int64(7), v.Violations[0].Line)
	require.Equal(t, "empty pattern", v.Violations[0].Message)
	require.Nil(t, v.Violations[0].Params)
}

func TestCodeOwnersValidation_AddfAtLine(t *testing.T) {
	var v CodeOwnersValidation
	v.AddfAtLine(enum.CodeOwnerViolationCodeUserNotFound, 3, "user %q not found", "a@b.c")

	require.Len(t, v.Violations, 1)
	require.Equal(t, int64(3), v.Violations[0].Line)
	require.Equal(t, `user "a@b.c" not found`, v.Violations[0].Message)
	require.Equal(t, []any{"a@b.c"}, v.Violations[0].Params)
}

func TestCodeOwnersViolation_LineOmittedWhenZero(t *testing.T) {
	var noLine CodeOwnersValidation
	noLine.Add(enum.CodeOwnerViolationCodePatternEmpty, "x")
	raw, err := json.Marshal(noLine)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"line"`)

	var withLine CodeOwnersValidation
	withLine.AddAtLine(enum.CodeOwnerViolationCodePatternEmpty, 4, "x")
	raw, err = json.Marshal(withLine)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"line":4`)
}
