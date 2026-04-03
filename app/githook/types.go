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

package githook

import (
	"errors"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Payload defines the payload that's send to git via environment variables.
type Payload struct {
	BaseURL       string
	RepoID        int64
	PrincipalID   int64
	RequestID     string
	Disabled      bool
	Internal      bool // Deprecated: Use OperationType instead
	OperationType enum.GitOpType
}

func (p Payload) Validate() error {
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

func GetInputBaseFromPayload(p Payload) types.GithookInputBase {
	// For backward compatibility: if OperationType is not set, infer from Internal field
	opType := p.OperationType
	if opType == "" {
		if p.Internal {
			opType = enum.GitOpTypeAPIContent
		} else {
			opType = enum.GitOpTypeGitPush
		}
	}

	return types.GithookInputBase{
		RepoID:        p.RepoID,
		PrincipalID:   p.PrincipalID,
		Internal:      opType != enum.GitOpTypeGitPush, // Maintain backward compatibility
		OperationType: opType,
	}
}
