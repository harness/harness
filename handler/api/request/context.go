// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package request

// https://github.com/kubernetes/apiserver/blob/master/pkg/endpoints/request/context.go

import (
	"context"

	"github.com/drone/drone/core"
)

type key int

const (
	userKey key = iota
	permKey
	repoKey
)

// WithUser returns a copy of parent in which the user value is set
func WithUser(parent context.Context, user *core.User) context.Context {
	return context.WithValue(parent, userKey, user)
}

// UserFrom returns the value of the user key on the ctx
func UserFrom(ctx context.Context) (*core.User, bool) {
	user, ok := ctx.Value(userKey).(*core.User)
	return user, ok
}

// WithPerm returns a copy of parent in which the perm value is set
func WithPerm(parent context.Context, perm *core.Perm) context.Context {
	return context.WithValue(parent, permKey, perm)
}

// PermFrom returns the value of the perm key on the ctx
func PermFrom(ctx context.Context) (*core.Perm, bool) {
	perm, ok := ctx.Value(permKey).(*core.Perm)
	return perm, ok
}

// WithRepo returns a copy of parent in which the repo value is set
func WithRepo(parent context.Context, repo *core.Repository) context.Context {
	return context.WithValue(parent, repoKey, repo)
}

// RepoFrom returns the value of the repo key on the ctx
func RepoFrom(ctx context.Context) (*core.Repository, bool) {
	repo, ok := ctx.Value(repoKey).(*core.Repository)
	return repo, ok
}
