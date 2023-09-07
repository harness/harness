// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package file

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	// File represents the raw file contents in the
	// version control system.
	File struct {
		Data []byte
	}

	// FileService provides access to contents of files in
	// the SCM provider. Today, this is gitness but it should
	// be extendible to any SCM provider.
	// The plan is for all remote repos to be pointers inside gitness
	// so a repo entry would always exist. If this changes, the interface
	// can be updated.
	FileService interface {
		// path is the path in the repo to read
		// ref is the git ref for the repository e.g. refs/heads/master
		Get(ctx context.Context, repo *types.Repository, path, ref string) (*File, error)
	}
)
