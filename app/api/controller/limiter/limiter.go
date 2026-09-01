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

package limiter

import (
	"context"

	"github.com/harness/gitness/errors"
)

var ErrMaxNumReposReached = errors.New("maximum number of repositories reached")
var ErrMaxRepoSizeReached = errors.New("maximum size of repository reached")
var ErrMaxTotalStorageReached = errors.New("maximum total storage reached")

// ErrRepoSizeSoftLimitReached indicates the repository grew past its advisory size
// limit. Unlike the errors above it is not a rejection: callers are expected to
// surface it to the user as a warning and let the operation proceed.
var ErrRepoSizeSoftLimitReached = errors.New("repository size soft limit reached")

// ResourceLimiter is an interface for managing resource limitation.
type ResourceLimiter interface {
	// RepoCount allows the creation of a specified number of repositories.
	RepoCount(ctx context.Context, spaceID int64, count int) error

	// RepoSize allows repository growth up to a limit for the given repoID.
	// It returns ErrMaxRepoSizeReached when the repository exceeds its hard limit
	// and ErrRepoSizeSoftLimitReached when it only exceeds the advisory one.
	RepoSize(ctx context.Context, repoID int64) error

	// RootSpaceStorage allows storage growth of a space's root space up to its
	// limit, returning ErrMaxTotalStorageReached once it is exceeded. spaceID may
	// be any space in the hierarchy; implementations resolve its root space.
	// additionalKiB is the incremental size (KiB) the operation would add.
	RootSpaceStorage(ctx context.Context, spaceID int64, additionalKiB int64) error
}

var _ ResourceLimiter = Unlimited{}

type Unlimited struct {
}

// NewResourceLimiter creates a new instance of ResourceLimiter.
func NewResourceLimiter() ResourceLimiter {
	return Unlimited{}
}

func (Unlimited) RepoCount(context.Context, int64, int) error {
	return nil
}

func (Unlimited) RepoSize(context.Context, int64) error {
	return nil
}

func (Unlimited) RootSpaceStorage(context.Context, int64, int64) error {
	return nil
}
