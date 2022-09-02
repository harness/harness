// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

type Authorizer interface {
	CheckForAccess(principalType PrincipalType, principalId string, resource Resource, permission Permission) error
	CheckForAccessAll(principalType PrincipalType, principalId string, permissionChecks ...PermissionCheck) error
}

type PermissionCheck struct {
	Resource   Resource
	Permission Permission
}

type Resource struct {
	Type       ResourceType
	Identifier string
}

type ResourceType string

const (
	ResourceType_Space      ResourceType = "SPACE"
	ResourceType_Repository ResourceType = "REPOSITORY"
	//   ResourceType_Branch ResourceType = "BRANCH"
)

type Permission string

const (
	// ----- SPACE -----
	Permission_Space_Create Permission = "space_create"
	Permission_Space_View   Permission = "space_view"
	Permission_Space_Edit   Permission = "space_edit"

	// ----- REPOSITORY -----
	Permission_Repository_Create Permission = "repository_create"
	Permission_Repository_View   Permission = "repository_view"
	Permission_Repository_Edit   Permission = "repository_edit"

	// ----- BRANCH -----
	// Permission_Branch_Create ResourcePermission = "branch_create"
	// Permission_Branch_View   ResourcePermission = "branch_view"
	// Permission_Branch_Edit   ResourcePermission = "branch_edit"
)

type PrincipalType string

const (
	PrincipalType_User   PrincipalType = "USER"
	PrincipalType_ApiKey PrincipalType = "API_KEY"
)
