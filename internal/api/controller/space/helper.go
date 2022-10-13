// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"strconv"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

func findSpaceFromRef(ctx context.Context, spaceStore store.SpaceStore, spaceRef string) (*types.Space, error) {
	// check if ref is spaceId - ASSUMPTION: digit only is no valid space name
	id, err := strconv.ParseInt(spaceRef, 10, 64)
	if err == nil {
		return spaceStore.Find(ctx, id)
	}

	return spaceStore.FindByPath(ctx, spaceRef)
}
