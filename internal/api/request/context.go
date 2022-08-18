// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

// This pattern was inpired by the kubernetes request context package.
// https://github.com/kubernetes/apiserver/blob/master/pkg/endpoints/request/context.go

import (
	"context"

	"github.com/harness/scm/types"
)

type key int

const (
	userKey key = iota
	projKey
)

// WithUser returns a copy of parent in which the user
// value is set
func WithUser(parent context.Context, v *types.User) context.Context {
	return context.WithValue(parent, userKey, v)
}

// UserFrom returns the value of the user key on the
// context.
func UserFrom(ctx context.Context) (*types.User, bool) {
	v, ok := ctx.Value(userKey).(*types.User)
	return v, ok
}
