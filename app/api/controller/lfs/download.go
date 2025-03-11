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

package lfs

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Download(ctx context.Context,
	session *auth.Session,
	repoRef string,
	oid string,
) (io.ReadCloser, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	_, err = c.lfsStore.Find(ctx, repo.ID, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to find the oid %q for the repo: %w", oid, err)
	}

	objPath := getLFSObjectPath(oid)
	file, err := c.blobStore.Download(ctx, objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from blobstore: %w", err)
	}

	return file, nil
}
