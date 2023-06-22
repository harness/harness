// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"

	gitea "code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
)

type Adapter struct {
}

func New() (Adapter, error) {
	// TODO: should be subdir of gitRoot? What is it being used for?
	setting.Git.HomePath = "home"

	err := gitea.InitSimple(context.Background())
	if err != nil {
		return Adapter{}, err
	}

	return Adapter{}, nil
}
