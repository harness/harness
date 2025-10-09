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
	Type       enum.ResourceType
	Identifier string
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
