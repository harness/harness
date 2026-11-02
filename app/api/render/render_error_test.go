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

package render

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/usererror"

	"github.com/stretchr/testify/assert"
)

func TestUserError_RetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		err    *usererror.Error
		expect string
	}{
		{"locked", usererror.ErrResourceLocked, "1"},
		{"unavailable", usererror.ErrServiceUnavailable, "5"},
		{"bad request", usererror.ErrBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			UserError(context.Background(), w, tt.err)

			assert.Equal(t, tt.err.Status, w.Code)
			assert.Equal(t, tt.expect, w.Header().Get("Retry-After"))
		})
	}
}
