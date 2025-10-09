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

package sharedrepo

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lineNumberIsEOF(t *testing.T) {
	assert.True(t, lineNumberEOF.IsEOF(), "lineNumberEOF should be EOF")
	assert.True(t, lineNumber(math.MaxInt64).IsEOF(), "lineNumberEOF should be EOF")
	assert.False(t, lineNumber(1).IsEOF(), "1 should not be EOF")
}

func Test_lineNumberString(t *testing.T) {
	assert.Equal(t, "eof", lineNumberEOF.String(), "lineNumberEOF should be 'eof'")
	assert.Equal(t, "eof", lineNumber(math.MaxInt64).String(), "math.MaxInt64 should be 'eof'")
	assert.Equal(t, "1", lineNumber(1).String(), "1 should be '1'")
}

func Test_parseLineNumber(t *testing.T) {
	tests := []struct {
		name    string
		arg     []byte
		wantErr string
		want    lineNumber
	}{
		{
			name:    "test empty",
			arg:     nil,
			wantErr: "line number can't be empty",
		},
		{
			name:    "test not a number",
			arg:     []byte("test"),
			wantErr: "unable to parse",
		},
		{
			name:    "test maxInt64+1 fails",
			arg:     []byte("9223372036854775808"),
			wantErr: "unable to parse",
		},
		{
			name:    "test smaller than 1",
			arg:     []byte("0"),
			wantErr: "line numbering starts at 1",
		},
		{
			name:    "test maxInt64 not allowed",
			arg:     []byte("9223372036854775807"),
			wantErr: "line numbering ends at 9223372036854775806",
		},
		{
			name: "test smallest valid number (1)",
			arg:  []byte("1"),
			want: 1,
		},
		{
			name: "test biggest valid number (maxInt64-1)",
			arg:  []byte("9223372036854775806"),
			want: 9223372036854775806,
		},
		{
			name: "test eof",
			arg:  []byte("eof"),
			want: lineNumberEOF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLineNumber(tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr, "error doesn't match expected.")
			} else {
				assert.Equal(t, tt.want, got, "parsed valued doesn't match expected")
			}
		})
	}
}
