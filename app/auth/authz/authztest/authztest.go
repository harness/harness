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

// Package authztest provides authz.Authorizer stubs for use in tests.
package authztest

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// AllowAuthorizer is a test authorizer that grants every permission check.
type AllowAuthorizer struct{}

func (AllowAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	return true, nil
}

func (AllowAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	return true, nil
}

func (AllowAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, permissionChecks ...types.PermissionCheck,
) ([]bool, error) {
	results := make([]bool, len(permissionChecks))
	for i := range results {
		results[i] = true
	}
	return results, nil
}

// DenyAuthorizer is a test authorizer that denies every permission check.
type DenyAuthorizer struct{}

func (DenyAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	return false, nil
}

func (DenyAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	return false, nil
}

func (DenyAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, permissionChecks ...types.PermissionCheck,
) ([]bool, error) {
	return make([]bool, len(permissionChecks)), nil
}

var (
	_ authz.Authorizer = AllowAuthorizer{}
	_ authz.Authorizer = DenyAuthorizer{}
)
