// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ResourceType represents the different types of resources that can be guarded with permissions.
type ResourceType string

const (
	ResourceTypeSpace ResourceType = "SPACE"
	ResourceTypeRepo  ResourceType = "REPOSITORY"
	//   ResourceType_Branch ResourceType = "BRANCH"
)

// Permission represents the available permissions.
type Permission string

const (
	/*
	   ----- SPACE -----
	*/
	PermissionSpaceCreate Permission = "space_create"
	PermissionSpaceView   Permission = "space_view"
	PermissionSpaceEdit   Permission = "space_edit"
	PermissionSpaceDelete Permission = "space_delete"
)

const (
	/*
		----- REPOSITORY -----
	*/
	PermissionRepoCreate Permission = "repository_create"
	PermissionRepoView   Permission = "repository_view"
	PermissionRepoEdit   Permission = "repository_edit"
	PermissionRepoDelete Permission = "repository_delete"
)

// PrincipalType represents the type of the entity requesting permission.
type PrincipalType string

const (
	// PrincipalTypeUser represents actions executed by a logged-in user.
	PrincipalTypeUser PrincipalType = "USER"
)
