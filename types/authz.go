// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "github.com/harness/gitness/types/enum"

// PermissionCheck represents a permission check.
type PermissionCheck struct {
	Scope      Scope
	Resource   Resource
	Permission enum.Permission
}

// Resource represents the resource of a permission check.
// Note: Keep the name empty in case access is requested for all resources of that type.
type Resource struct {
	Type enum.ResourceType
	Name string
}

// Scope represents the scope of a permission check
// Notes:
//   - In case the permission check is for resource REPO, keep repo empty (repo is resource, not scope)
//   - In case the permission check is for resource SPACE, SpacePath is an ancestor of the space (space is
//     resource, not scope)
//   - Repo isn't use as of now (will be useful once we add access control for repo child resources, e.g. branches).
type Scope struct {
	SpacePath string
	Repo      string
}
