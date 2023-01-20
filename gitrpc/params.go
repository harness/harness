// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

type Repository interface {
	GetGitUID() string
}

// CreateRPCReadParams creates base read parameters for gitrpc read operations.
// IMPORTANT: repo is assumed to be not nil!
func CreateRPCReadParams(repo Repository) ReadParams {
	return ReadParams{
		RepoUID: repo.GetGitUID(),
	}
}
