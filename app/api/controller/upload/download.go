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

package upload

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Download(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filePath string,
) (string, io.ReadCloser, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return "", nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	fileBucketPath := getFileBucketPath(repo.ID, filePath)

	signedURL, err := c.blobStore.GetSignedURL(ctx, fileBucketPath)
	if err != nil && !errors.Is(err, blob.ErrNotSupported) {
		return "", nil, fmt.Errorf("failed to get signed URL: %w", err)
	}

	if signedURL != "" {
		return signedURL, nil, nil
	}

	file, err := c.blobStore.Download(ctx, fileBucketPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download file from blobstore: %w", err)
	}

	return "", file, nil
}
