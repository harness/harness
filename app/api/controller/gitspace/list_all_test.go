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

package gitspace

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// fakeAuthorizer lets each test decide, per space path, whether Check should
// grant or deny access, so getAuthorizedSpaces can be exercised without a
// real permission store.
type fakeAuthorizer struct {
	allowed map[string]bool
}

func (f *fakeAuthorizer) Check(
	_ context.Context,
	_ *auth.Session,
	scope *types.Scope,
	_ *types.Resource,
	_ enum.Permission,
) (bool, error) {
	return f.allowed[scope.SpacePath], nil
}

func (f *fakeAuthorizer) CheckAll(
	_ context.Context,
	_ *auth.Session,
	_ ...types.PermissionCheck,
) (bool, error) {
	return false, nil
}

func (f *fakeAuthorizer) CheckMany(
	_ context.Context,
	_ *auth.Session,
	_ ...types.PermissionCheck,
) ([]bool, error) {
	return nil, nil
}

func TestGetAuthorizedSpaces_ExcludesDeniedSpaces(t *testing.T) {
	authorizer := &fakeAuthorizer{
		allowed: map[string]bool{
			"space-a": true,
			// space-b is left out of the map, so Check denies it.
		},
	}
	controller := &Controller{authorizer: authorizer}
	session := &auth.Session{
		Principal: types.Principal{ID: 1, UID: "user-1", Type: enum.PrincipalTypeUser},
	}

	result, err := controller.getAuthorizedSpaces(context.Background(), session, map[int64]string{
		1: "space-a",
		2: "space-b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result[1] {
		t.Errorf("space 1 (authorized) should be present in the result")
	}
	if result[2] {
		t.Errorf("space 2 (denied) should not be present in the result, got authorized")
	}
	if len(result) != 1 {
		t.Errorf("expected exactly 1 authorized space, got %d: %v", len(result), result)
	}
}
