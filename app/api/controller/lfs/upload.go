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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type UploadOut struct {
	ObjectPath string `json:"object_path"`
}

func (c *Controller) Upload(ctx context.Context,
	session *auth.Session,
	repoRef string,
	pointer Pointer,
	file io.Reader,
) (*UploadOut, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if file == nil {
		return nil, usererror.BadRequest("no file or content provided")
	}

	bufReader := bufio.NewReader(file)
	objPath := getLFSObjectPath(pointer.OId)

	err = c.blobStore.Upload(ctx, bufReader, objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	now := time.Now()
	object := &types.LFSObject{
		OID:       pointer.OId,
		Size:      pointer.Size,
		Created:   now.UnixMilli(),
		CreatedBy: session.Principal.ID,
		RepoID:    repo.ID,
	}
	// create the object in lfs store after successful upload to the blob store.
	err = c.lfsStore.Create(ctx, object)
	if err != nil && !errors.Is(err, store.ErrDuplicate) {
		return nil, fmt.Errorf("failed to find object: %w", err)
	}

	return &UploadOut{
		ObjectPath: objPath,
	}, nil
}
