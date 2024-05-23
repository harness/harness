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
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"

	"github.com/google/uuid"
)

// Result contains the information about the upload.
type Result struct {
	FilePath string `json:"file_path"`
}

const (
	fileNameFmt = "%s%s"
)

func (c *Controller) Upload(ctx context.Context,
	session *auth.Session,
	repoRef string,
	file io.Reader,
) (*Result, error) {
	// Permission check to see if the user in request has access to the repo.
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if file == nil {
		return nil, usererror.BadRequest("no file provided")
	}
	bufReader := bufio.NewReader(file)
	// Check if the file is an image or video
	extn, err := c.getFileExtension(bufReader)
	if err != nil {
		return nil, fmt.Errorf("failed to determine file type: %w", err)
	}

	identifier := uuid.New().String()
	fileName := fmt.Sprintf(fileNameFmt, identifier, extn)

	fileBucketPath := getFileBucketPath(repo.ID, fileName)
	err = c.blobStore.Upload(ctx, bufReader, fileBucketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	return &Result{
		FilePath: fileName,
	}, nil
}
