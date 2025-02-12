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

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
)

func (c *Controller) checkFileSizeLimit(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	// return if all new refs are nil refs
	allNilRefs := true
	for _, refUpdate := range in.RefUpdates {
		if refUpdate.New.IsNil() {
			continue
		}
		allNilRefs = false
		break
	}
	if allNilRefs {
		return nil
	}

	sizeLimit, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyFileSizeLimit,
		settings.DefaultFileSizeLimit,
	)
	if err != nil {
		return fmt.Errorf("failed to check settings for file size limit: %w", err)
	}
	if sizeLimit <= 0 {
		return nil
	}

	res, err := rgit.FindOversizeFiles(
		ctx,
		&git.FindOversizeFilesParams{
			RepoUID:       repo.GitUID,
			GitObjectDirs: in.Environment.AlternateObjectDirs,
			SizeLimit:     sizeLimit,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get file sizes: %w", err)
	}

	if len(res.FileInfos) > 0 {
		output.Error = ptr.String("Changes blocked by files exceeding the file size limit")
		printOversizeFiles(output, res.FileInfos, sizeLimit)
	}

	return nil
}
