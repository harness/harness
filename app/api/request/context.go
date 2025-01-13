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

package request

// This pattern was inpired by the kubernetes request context package.
// https://github.com/kubernetes/apiserver/blob/master/pkg/endpoints/request/context.go

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"

	"github.com/gliderlabs/ssh"
)

type key int

const (
	authSessionKey key = iota
	serviceAccountKey
	userKey
	spaceKey
	repoKey
	requestIDKey
)

// WithAuthSession returns a copy of parent in which the principal
// value is set.
func WithAuthSession(parent context.Context, v *auth.Session) context.Context {
	return context.WithValue(parent, authSessionKey, v)
}

// AuthSessionFrom returns the value of the principal key on the
// context.
func AuthSessionFrom(ctx context.Context) (*auth.Session, bool) {
	v, ok := ctx.Value(authSessionKey).(*auth.Session)
	return v, ok && v != nil
}

// PrincipalFrom returns the principal of the authsession.
func PrincipalFrom(ctx context.Context) (*types.Principal, bool) {
	v, ok := AuthSessionFrom(ctx)
	if !ok {
		return nil, false
	}

	return &v.Principal, true
}

// WithUser returns a copy of parent in which the user value is set.
func WithUser(parent context.Context, v *types.User) context.Context {
	return context.WithValue(parent, userKey, v)
}

// UserFrom returns the value of the user key on the
// context - ok is true iff a non-nile value existed.
func UserFrom(ctx context.Context) (*types.User, bool) {
	v, ok := ctx.Value(userKey).(*types.User)
	return v, ok && v != nil
}

// WithServiceAccount returns a copy of parent in which the service account value is set.
func WithServiceAccount(parent context.Context, v *types.ServiceAccount) context.Context {
	return context.WithValue(parent, serviceAccountKey, v)
}

// ServiceAccountFrom returns the value of the service account key on the
// context - ok is true iff a non-nile value existed.
func ServiceAccountFrom(ctx context.Context) (*types.ServiceAccount, bool) {
	v, ok := ctx.Value(serviceAccountKey).(*types.ServiceAccount)
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

// WithRequestID returns a copy of parent in which the request id value is set.
func WithRequestID(parent context.Context, v string) context.Context {
	return context.WithValue(parent, requestIDKey, v)
}

// RequestIDFrom returns the value of the request ID key on the
// context - ok is true iff a non-empty value existed.
//
//nolint:revive // need to emphasize that it's the request id we are retrieving.
func RequestIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey).(string)
	return v, ok && v != ""
}

func WithRequestIDSSH(parent ssh.Context, v string) {
	ssh.Context.SetValue(parent, requestIDKey, v)
}
