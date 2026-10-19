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

package sharedrepo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/git/hook"
)

type Runner interface {
	// Run makes changes in a repository through sharedrepo.SharedRepo.
	// Even if refUpdater is nil or its list of references are empty,
	// still new objects could be introduced to the original repository.
	Run(
		ctx context.Context,
		refUpdater *hook.RefUpdater,
		tmpDir, repoPath string,
		fn func(s *SharedRepo) error,
		alternates ...string,
	) error
}

type ReadOnlyRunner interface {
	// Run executes the provided function in a sharedrepo.SharedRepo.
	// Any changes made to the repository would be discarded.
	Run(
		ctx context.Context,
		tmpDir, repoPath string,
		fn func(s *SharedRepo) error,
		alternates ...string,
	) error
}

type DefaultRunner struct{}

// Run is used to run the provided function inside a shared repository.
// If the provided hook.RefUpdater is not nil it will be used to update the reference.
// Inside the provided inline function there should be a call to initialize the ref updater.
func (DefaultRunner) Run(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	tmpDir, repoPath string,
	fn func(s *SharedRepo) error,
	alternates ...string,
) error {
	s, err := NewSharedRepo(tmpDir, repoPath)
	if err != nil {
		return err
	}

	defer s.Close(ctx)

	if err := s.Init(ctx, alternates...); err != nil {
		return fmt.Errorf("failed to init shared repository: %w", err)
	}

	// The refUpdater.Init must be called within the fn (if refUpdater != nil), otherwise the refUpdater.Pre will fail.
	if err := fn(s); err != nil {
		return fmt.Errorf("error in external function working in sharedrepo: %w", err)
	}

	alternate := s.Directory() + "/objects"

	if err := refUpdater.Pre(ctx, alternate); err != nil {
		return fmt.Errorf("pre-receive hook failed: %w", err)
	}

	// Objects are packed only after the pre-receive hook has approved the update: The hook reads
	// the new objects through the alternate object directory, so it doesn't benefit from the pack
	// file, and a rejected update doesn't need one at all.
	if err := s.PackObjects(ctx, packObjectsThreshold); err != nil {
		return fmt.Errorf("failed to pack objects: %w", err)
	}

	if err := s.MoveObjects(ctx); err != nil {
		return fmt.Errorf("failed to move objects: %w", err)
	}

	if err := refUpdater.UpdateRef(ctx); err != nil {
		return fmt.Errorf("failed to update reference: %w", err)
	}

	if err := refUpdater.Post(ctx, alternate); err != nil {
		return fmt.Errorf("post-receive hook failed: %w", err)
	}

	return nil
}

type DefaultReadOnlyRunner struct{}

// Run is used to run the provided function inside a shared repository.
func (DefaultReadOnlyRunner) Run(
	ctx context.Context,
	tmpDir, repoPath string,
	fn func(s *SharedRepo) error,
	alternates ...string,
) error {
	s, err := NewSharedRepo(tmpDir, repoPath)
	if err != nil {
		return err
	}

	defer s.Close(ctx)

	if err := s.Init(ctx, alternates...); err != nil {
		return fmt.Errorf("failed to init shared repository: %w", err)
	}

	if err := fn(s); err != nil {
		return fmt.Errorf("error in external function working in sharedrepo: %w", err)
	}

	return nil
}
