// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package rpc2

// Copyright 2019 Drone.IO Inc. All rights reserved.
import (
	"github.com/drone/drone/core"
	"github.com/drone/drone/operator/manager"
)

// details provides the runner with the build details and
// includes all environment data required to execute the build.
type details struct {
	*manager.Context
	Netrc *core.Netrc `json:"netrc"`
	Repo  *repositroy `json:"repository"`
}

// repository wraps a repository object to include the secret
// when the repository is marshaled to json.
type repositroy struct {
	*core.Repository
	Secret string `json:"secret"`
}
