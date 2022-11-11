// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"
	"io"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/types"
)

// GetBlob returns the blob for the given object sha.
// Note: sha is the object sha.
func (g Adapter) GetBlob(ctx context.Context, repoPath string, sha string, sizeLimit int64) (*types.Blob, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	giteaBlob, err := giteaRepo.GetBlob(sha)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting blob '%s'", sha)
	}

	reader, err := giteaBlob.DataAsync()
	if err != nil {
		return nil, processGiteaErrorf(err, "error opening data for blob '%s'", sha)
	}

	returnSize := giteaBlob.Size()
	if sizeLimit > 0 && returnSize > sizeLimit {
		returnSize = sizeLimit
	}

	// TODO: ensure it doesn't fail because buff has exact size of bytes required
	buff := make([]byte, returnSize)
	_, err = io.ReadAtLeast(reader, buff, int(returnSize))
	if err != nil {
		return nil, fmt.Errorf("error reading data from blob '%s': %w", sha, err)
	}

	return &types.Blob{
		Size:    giteaBlob.Size(),
		Content: buff,
	}, nil
}
