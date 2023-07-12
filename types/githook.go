// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"errors"
)

// GithookPayload defines the GithookPayload the githook binary is initiated with when executing the git hooks.
type GithookPayload struct {
	BaseURL     string
	RepoID      int64
	PrincipalID int64
	RequestID   string
	Disabled    bool
}

func (p *GithookPayload) Validate() error {
	if p == nil {
		return errors.New("payload is empty")
	}

	// skip further validation if githook is disabled
	if p.Disabled {
		return nil
	}

	if p.BaseURL == "" {
		return errors.New("payload doesn't contain a base url")
	}
	if p.PrincipalID <= 0 {
		return errors.New("payload doesn't contain a principal id")
	}
	if p.RepoID <= 0 {
		return errors.New("payload doesn't contain a repo id")
	}

	return nil
}
