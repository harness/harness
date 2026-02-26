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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/blob"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	lfsTempPathFormat = "lfs/tmp/%s"
)

type UploadOut struct {
	ObjectPath string `json:"object_path"` //nolint:tagliatelle
}

// hashingReader wraps an io.Reader and calculates a hash while reading.
type hashingReader struct {
	reader io.Reader
	hasher hash.Hash
}

func newHashingReader(r io.Reader) *hashingReader {
	return &hashingReader{
		reader: r,
		hasher: sha256.New(),
	}
}

func (h *hashingReader) Read(p []byte) (int, error) {
	n, err := h.reader.Read(p)
	if n > 0 {
		h.hasher.Write(p[:n])
	}
	return n, err
}

func (h *hashingReader) Sum() string {
	return hex.EncodeToString(h.hasher.Sum(nil))
}

func (c *Controller) Upload(ctx context.Context,
	session *auth.Session,
	repoRef string,
	pointer Pointer,
	file io.Reader,
) (*UploadOut, error) {
	var additionalAllowedRepoStates = []enum.RepoState{enum.RepoStateMigrateGitPush}
	repoCore, err := c.getRepoCheckAccessAndSetting(ctx, session, repoRef,
		enum.PermissionRepoPush, additionalAllowedRepoStates...)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if file == nil {
		return nil, usererror.BadRequest("No file or content provided")
	}

	_, err = c.lfsStore.Find(ctx, repoCore.ID, pointer.OId)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to check if object exists: %w", err)
	}
	if err == nil {
		return nil, usererror.Conflict("LFS object already exists and cannot be modified")
	}

	expectedHash := strings.TrimPrefix(pointer.OId, "sha256:")

	// Generate a unique temp path for staging the upload
	tempPath := fmt.Sprintf(lfsTempPathFormat, uuid.NewString())
	finalPath := getLFSObjectPath(pointer.OId)

	// Stream the content to temp path while calculating hash
	limitedReader := io.LimitReader(file, pointer.Size)
	hashReader := newHashingReader(limitedReader)

	err = c.blobStore.Upload(ctx, hashReader, tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to temp path: %w", err)
	}

	calculatedHash := hashReader.Sum()
	if calculatedHash != expectedHash {
		if deleteErr := c.blobStore.Delete(ctx, tempPath); deleteErr != nil {
			if !errors.Is(deleteErr, blob.ErrNotFound) {
				log.Ctx(ctx).Warn().Err(deleteErr).
					Str("temp_path", tempPath).
					Msg("failed to delete temp file after hash mismatch")
			}
		}
		return nil, usererror.BadRequest("content hash doesn't match provided OID")
	}

	err = c.blobStore.Move(ctx, tempPath, finalPath)
	if err != nil {
		if deleteErr := c.blobStore.Delete(ctx, tempPath); deleteErr != nil {
			if !errors.Is(deleteErr, blob.ErrNotFound) {
				log.Ctx(ctx).Warn().Err(deleteErr).
					Str("temp_path", tempPath).
					Msg("failed to delete temp file after move failure")
			}
		}
		return nil, fmt.Errorf("failed to move file to final path: %w", err)
	}

	now := time.Now()
	object := &types.LFSObject{
		OID:       pointer.OId,
		Size:      pointer.Size,
		Created:   now.UnixMilli(),
		CreatedBy: session.Principal.ID,
		RepoID:    repoCore.ID,
	}

	// create the object in lfs store after successful upload to the blob store.
	err = c.lfsStore.Create(ctx, object)
	if err != nil && !errors.Is(err, store.ErrDuplicate) {
		return nil, fmt.Errorf("failed to create object: %w", err)
	}

	return &UploadOut{
		ObjectPath: finalPath,
	}, nil
}
