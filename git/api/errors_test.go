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
	"errors"
	"testing"

	gitnesserrors "github.com/harness/gitness/errors"

	"github.com/stretchr/testify/assert"
)

func TestProcessGitErrorf(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus gitnesserrors.Status
	}{
		{
			name: "unknown revision",
			err: errors.New(
				"fatal: ambiguous argument 'refs/heads/undefined': unknown revision or path not in the working tree.",
			),
			wantStatus: gitnesserrors.StatusNotFound,
		},
		{
			name:       "ambiguous argument only",
			err:        errors.New("fatal: ambiguous argument 'HEAD': unknown revision or path not in the working tree."),
			wantStatus: gitnesserrors.StatusNotFound,
		},
		{
			name:       "no such file or directory - existing mapping unaffected",
			err:        errors.New("open /data/repo.git: no such file or directory"),
			wantStatus: gitnesserrors.StatusNotFound,
		},
		{
			name:       "reference already exists - existing mapping unaffected",
			err:        errors.New("fatal: reference already exists"),
			wantStatus: gitnesserrors.StatusConflict,
		},
		{
			name:       "unrecognised error still falls back to internal",
			err:        errors.New("fatal: some completely unrelated git failure"),
			wantStatus: gitnesserrors.StatusInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processGitErrorf(tt.err, "failed to run git command")
			assert.Equal(t, tt.wantStatus, gitnesserrors.AsStatus(got))
		})
	}
}
