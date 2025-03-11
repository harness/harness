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

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
)

func (c *Controller) checkLFSObjects(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	// return if all new refs are nil refs
	if isAllRefDeletions(in.RefUpdates) {
		return nil
	}

	res, err := rgit.ListLFSPointers(ctx,
		&git.ListLFSPointersParams{
			ReadParams: git.ReadParams{
				RepoUID:             repo.GitUID,
				AlternateObjectDirs: in.Environment.AlternateObjectDirs,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list lfs pointers: %w", err)
	}

	if len(res.LFSInfos) == 0 {
		return nil
	}

	oids := make([]string, len(res.LFSInfos))
	for i := range res.LFSInfos {
		oids[i] = res.LFSInfos[i].OID
	}

	existingObjs, err := c.lfsStore.FindMany(ctx, in.RepoID, oids)
	if err != nil {
		return fmt.Errorf("failed to find lfs objects: %w", err)
	}

	//nolint:lll
	if len(existingObjs) != len(oids) {
		output.Error = ptr.String(
			"Changes blocked by missing lfs objects. Please try `git lfs push --all` or check if LFS is setup properly.")
		return nil
	}

	return nil
}

func isAllRefDeletions(refUpdates []hook.ReferenceUpdate) bool {
	for _, refUpdate := range refUpdates {
		if !refUpdate.New.IsNil() {
			return false
		}
	}

	return true
}
