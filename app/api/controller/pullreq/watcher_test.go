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

package pullreq

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
)

func TestWatchUnwatch_AnonymousIsRejected(t *testing.T) {
	t.Parallel()

	// The anonymous check happens before any repo lookup, so no dependencies are needed.
	c := &Controller{}
	session := &auth.Session{Principal: auth.AnonymousPrincipal}

	if err := c.Watch(context.Background(), session, "space/repo", 1); !errors.Is(err, usererror.ErrUnauthorized) {
		t.Errorf("Watch: got %v, want %v", err, usererror.ErrUnauthorized)
	}
	if err := c.Unwatch(context.Background(), session, "space/repo", 1); !errors.Is(err, usererror.ErrUnauthorized) {
		t.Errorf("Unwatch: got %v, want %v", err, usererror.ErrUnauthorized)
	}
}
