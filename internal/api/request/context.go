// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

// This pattern was inpired by the kubernetes request context package.
// https://github.com/kubernetes/apiserver/blob/master/pkg/endpoints/request/context.go

import (
	"context"

	"github.com/harness/gitness/types"
)

type key int

const (
	userKey key = iota
	spaceKey
	repoKey
)

// WithUser returns a copy of parent in which the user
// value is set.
func WithUser(parent context.Context, v *types.User) context.Context {
	return context.WithValue(parent, userKey, v)
}

// UserFrom returns the value of the user key on the
// context.
func UserFrom(ctx context.Context) (*types.User, bool) {
	v, ok := ctx.Value(userKey).(*types.User)
	return v, ok && v != nil
}

// WithSpace returns a copy of parent in which the space value is set.
func WithSpace(parent context.Context, v *types.Space) context.Context {
	return context.WithValue(parent, spaceKey, v)
}

// SpaceFrom returns the value of the space key on the
// context - ok is true iff a non-nile value existed.
func SpaceFrom(ctx context.Context) (*types.Space, bool) {
	v, ok := ctx.Value(spaceKey).(*types.Space)
	return v, ok && v != nil
}

// WithRepo returns a copy of parent in which the repo value is set.
func WithRepo(parent context.Context, v *types.Repository) context.Context {
	return context.WithValue(parent, repoKey, v)
}

// RepoFrom returns the value of the repo key on the
// context - ok is true iff a non-nile value existed.
func RepoFrom(ctx context.Context) (*types.Repository, bool) {
	v, ok := ctx.Value(repoKey).(*types.Repository)
	return v, ok && v != nil
}
