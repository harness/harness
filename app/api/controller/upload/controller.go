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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gabriel-vasile/mimetype"
)

const (
	MaxFileSize       = 10 << 20 // 10 MB file limit set in Handler
	fileBucketPathFmt = "uploads/%d/%s"
	peekBytes         = 512
)

var supportedFileTypes = map[string]struct{}{
	"image": {},
	"video": {},
}

type Controller struct {
	authorizer authz.Authorizer
	repoStore  store.RepoStore
	blobStore  blob.Store
}

func NewController(authorizer authz.Authorizer,
	repoStore store.RepoStore,
	blobStore blob.Store,
) *Controller {
	return &Controller{
		authorizer: authorizer,
		repoStore:  repoStore,
		blobStore:  blobStore,
	}
}
func (c *Controller) getRepoCheckAccess(ctx context.Context,
	session *auth.Session,
	repoRef string,
	permission enum.Permission,
) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, permission); err != nil {
		return nil, fmt.Errorf("failed to verify authorization: %w", err)
	}

	return repo, nil
}

func (c *Controller) getFileExtension(file *bufio.Reader) (string, error) {
	buf, err := file.Peek(peekBytes)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Example: mType.String() = image/png
	// Splitting on "/" and taking the first element of the slice
	// will give us the file type.
	mType := mimetype.Detect(buf)
	if _, ok := supportedFileTypes[strings.Split(mType.String(), "/")[0]]; !ok {
		return "",
			usererror.BadRequestf(
				"only image and video files are supported, uploaded file is of type %s",
				mType.String())
	}

	return mType.Extension(), nil
}

func getFileBucketPath(repoID int64, fileName string) string {
	return fmt.Sprintf(fileBucketPathFmt, repoID, fileName)
}
