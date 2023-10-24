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
	Internal    bool // Internal calls comer for the Gitness, and external calls are direct git pushes.
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
