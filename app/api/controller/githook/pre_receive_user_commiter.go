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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
)

func (c *Controller) userCommiterMatchCheck(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	in types.GithookPreReceiveInput,
	principalEmail string,
	output *hook.Output,
) error {
	userCommiterMatch, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyUserCommiterMatch,
		settings.DefaultUserCommiterMatch,
	)
	if err != nil {
		return fmt.Errorf("failed to check settings for user commiter match: %w", err)
	}
	if !userCommiterMatch {
		return nil
	}

	invalidShaEmailMap := make(map[string]string)

	// block any push that contains commits not committed by the user
	for _, refUpdate := range in.RefUpdates {
		baseSHA, fallbackAvailable, err := GetBaseSHAForRefUpdate(
			ctx,
			rgit,
			repo,
			in.Environment,
			in.RefUpdates,
			refUpdate,
		)
		if err != nil {
			return fmt.Errorf("failed to get fallback sha: %w", err)
		}
		// the default branch doesn't exist yet
		if !fallbackAvailable {
			baseSHA = sha.None
		}

		shaEmailMap, err := c.git.GetBranchCommiterEmails(ctx, &git.GetBranchCommitterEmailsParams{
			ReadParams: git.ReadParams{
				RepoUID:             repo.GitUID,
				AlternateObjectDirs: in.Environment.AlternateObjectDirs,
			},
			BaseSHA: baseSHA,
			RevSHA:  refUpdate.New,
		})
		if err != nil {
			return fmt.Errorf("failed to get commiter emails for %s: %w", refUpdate.Ref, err)
		}

		for sha, email := range shaEmailMap {
			if email != principalEmail {
				invalidShaEmailMap[sha] = email
			}
		}
	}

	if len(invalidShaEmailMap) > 0 {
		output.Error = ptr.String(usererror.ErrCommiterUserMismatch.Error())
		printUserCommiterMismatch(output, invalidShaEmailMap)
	}

	return nil
}
