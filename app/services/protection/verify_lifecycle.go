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

package protection

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	RefChangeVerifier interface {
		RefChangeVerify(ctx context.Context, in RefChangeVerifyInput) ([]types.RuleViolations, error)
	}

	RefChangeVerifyInput struct {
		ResolveUserGroupID func(ctx context.Context, userGroupIDs []int64) ([]int64, error)
		Actor              *types.Principal
		AllowBypass        bool
		IsRepoOwner        bool
		Repo               *types.Repository
		RefAction          RefAction
		RefType            RefType
		RefNames           []string
	}

	RefType int

	RefAction int

	DefLifecycle struct {
		CreateForbidden      bool `json:"create_forbidden,omitempty"`
		DeleteForbidden      bool `json:"delete_forbidden,omitempty"`
		UpdateForbidden      bool `json:"update_forbidden,omitempty"`
		UpdateForceForbidden bool `json:"update_force_forbidden,omitempty"`
	}
)

const (
	RefTypeRaw RefType = iota
	RefTypeBranch
	RefTypeTag
)

const (
	RefActionCreate RefAction = iota
	RefActionDelete
	RefActionUpdate
	RefActionUpdateForce
)

// ensures that the DefLifecycle type implements Sanitizer and RefChangeVerifier interfaces.
var (
	_ Sanitizer         = (*DefLifecycle)(nil)
	_ RefChangeVerifier = (*DefLifecycle)(nil)
)

const (
	codeLifecycleCreate      = "lifecycle.create"
	codeLifecycleDelete      = "lifecycle.delete"
	codeLifecycleUpdate      = "lifecycle.update"
	codeLifecycleUpdateForce = "lifecycle.update.force"
)

func (v *DefLifecycle) RefChangeVerify(_ context.Context, in RefChangeVerifyInput) ([]types.RuleViolations, error) {
	var violations types.RuleViolations

	switch in.RefAction {
	case RefActionCreate:
		if v.CreateForbidden {
			violations.Addf(codeLifecycleCreate,
				"Creation of branch %q is not allowed.", in.RefNames[0])
		}
	case RefActionDelete:
		if v.DeleteForbidden {
			violations.Addf(codeLifecycleDelete,
				"Delete of branch %q is not allowed.", in.RefNames[0])
		}
	case RefActionUpdate:
		if v.UpdateForbidden {
			violations.Addf(codeLifecycleUpdate,
				"Push to branch %q is not allowed. Please use pull requests.", in.RefNames[0])
		}
	case RefActionUpdateForce:
		if v.UpdateForceForbidden {
			violations.Addf(codeLifecycleUpdateForce,
				"Force push to branch %q is not allowed. Please use pull requests.", in.RefNames[0])
		}
	}

	if len(violations.Violations) > 0 {
		return []types.RuleViolations{violations}, nil
	}

	return nil, nil
}

func (*DefLifecycle) Sanitize() error {
	return nil
}
