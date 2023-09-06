// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package commit

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	// CommitService provides access to commit information via
	// the SCM provider. Today, this is gitness but it can
	// be extendible to any SCM provider.
	//
	// Arguments:
	//		repo: the repo to read content from
	// 		ref: the ref to fetch the commit from, eg refs/heads/master
	CommitService interface {
		FindRef(ctx context.Context, repo *types.Repository, ref string) (*types.Commit, error)
	}
)
