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

type Content struct {
	Data io.ReadCloser
	Size int64
}

func (c *Content) Read(p []byte) (n int, err error) {
	return c.Data.Read(p)
}

func (c *Content) Close() error {
	return c.Data.Close()
}

func (c *Controller) Download(ctx context.Context,
	session *auth.Session,
	repoRef string,
	oid string,
) (*Content, error) {
	repo, err := c.getRepoCheckAccessAndSetting(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	return c.DownloadNoAuth(ctx, repo.ID, oid)
}

func (c *Controller) DownloadNoAuth(
	ctx context.Context,
	repoID int64,
	oid string,
) (*Content, error) {
	obj, err := c.lfsStore.Find(ctx, repoID, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to find the oid %q for the repo: %w", oid, err)
	}

	objPath := getLFSObjectPath(oid)
	file, err := c.blobStore.Download(ctx, objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from blobstore: %w", err)
	}

	return &Content{
		Data: file,
		Size: obj.Size,
	}, nil
}
