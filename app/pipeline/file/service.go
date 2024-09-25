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

	// Service provides access to contents of files in
	// the SCM provider. Today, this is Harness but it should
	// be extendible to any SCM provider.
	// The plan is for all remote repos to be pointers inside Harness
	// so a repo entry would always exist. If this changes, the interface
	// can be updated.
	Service interface {
		// path is the path in the repo to read
		// ref is the git ref for the repository e.g. refs/heads/master
		Get(ctx context.Context, repo *types.Repository, path, ref string) (*File, error)
	}
)
