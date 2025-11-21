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

// Run is helper function used to run the provided function inside a shared repository.
// If the provided hook.RefUpdater is not nil it will be used to update the reference.
// Inside the provided inline function there should be a call to initialize the ref updater.
// If the provided hook.RefUpdater is nil the entire operation is a read-only.
func Run(
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
		return err
	}

	// The refUpdater.Init must be called within the fn (if refUpdater != nil), otherwise the refUpdater.Pre will fail.
	if err := fn(s); err != nil {
		return err
	}

	if refUpdater == nil {
		return nil
	}

	alternate := s.Directory() + "/objects"

	if err := refUpdater.Pre(ctx, alternate); err != nil {
		return fmt.Errorf("pre-receive hook failed: %w", err)
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
