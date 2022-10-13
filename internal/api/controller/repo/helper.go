// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"strconv"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

func findRepoFromRef(ctx context.Context, repoStore store.RepoStore, repoRef string) (*types.Repository, error) {
	// check if ref is repoId - ASSUMPTION: digit only is no valid repo name
	id, err := strconv.ParseInt(repoRef, 10, 64)
	if err == nil {
		return repoStore.Find(ctx, id)
	}

	return repoStore.FindByPath(ctx, repoRef)
}
