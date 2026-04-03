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

package usererror

import (
	"context"
	"net/http"
	"testing"

	"github.com/harness/gitness/errors"
)

func TestError(t *testing.T) {
	got, want := ErrNotFound.Message, ErrNotFound.Message
	if got != want {
		t.Errorf("Want error string %q, got %q", got, want)
	}
}

func TestTranslateUnprocessableEntity(t *testing.T) {
	err := errors.UnprocessableEntityf("pre-receive hook blocked reference update: %q", "blocked by protection rules")
	ctx := context.Background()

	result := Translate(ctx, err)

	if result.Status != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, result.Status)
	}

	expectedMsg := `pre-receive hook blocked reference update: "blocked by protection rules"`
	if result.Message != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, result.Message)
	}
}
