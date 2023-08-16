// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ResourceType represents the different types of resources that can be guarded with permissions.
type ResourceType string

const (
	ResourceTypeSpace          ResourceType = "SPACE"
	ResourceTypeRepo           ResourceType = "REPOSITORY"
	ResourceTypeUser           ResourceType = "USER"
	ResourceTypeServiceAccount ResourceType = "SERVICEACCOUNT"
	ResourceTypeService        ResourceType = "SERVICE"
	ResourceTypePipeline       ResourceType = "PIPELINE"
	ResourceTypeSecret         ResourceType = "SECRET"
	//   ResourceType_Branch ResourceType = "BRANCH"
)

// Permission represents the different types of permissions a principal can have.
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
	PermissionRepoView   Permission = "repo_view"
	PermissionRepoEdit   Permission = "repo_edit"
	PermissionRepoDelete Permission = "repo_delete"
	PermissionRepoPush   Permission = "repo_push"
)

const (
	/*
		----- USER -----
	*/
	PermissionUserCreate    Permission = "user_create"
	PermissionUserView      Permission = "user_view"
	PermissionUserEdit      Permission = "user_edit"
	PermissionUserDelete    Permission = "user_delete"
	PermissionUserEditAdmin Permission = "user_editAdmin"
)

const (
	/*
		----- SERVICE ACCOUNT -----
	*/
	PermissionServiceAccountCreate Permission = "serviceaccount_create"
	PermissionServiceAccountView   Permission = "serviceaccount_view"
	PermissionServiceAccountEdit   Permission = "serviceaccount_edit"
	PermissionServiceAccountDelete Permission = "serviceaccount_delete"
)

const (
	/*
		----- SERVICE -----
	*/
	PermissionServiceCreate    Permission = "service_create"
	PermissionServiceView      Permission = "service_view"
	PermissionServiceEdit      Permission = "service_edit"
	PermissionServiceDelete    Permission = "service_delete"
	PermissionServiceEditAdmin Permission = "service_editAdmin"
)

const (
	/*
		----- PIPELINE -----
	*/
	PermissionPipelineView    Permission = "pipeline_view"
	PermissionPipelineEdit    Permission = "pipeline_edit"
	PermissionPipelineDelete  Permission = "pipeline_delete"
	PermissionPipelineExecute Permission = "pipeline_execute"
)

const (
	/*
		----- SECRET -----
	*/
	PermissionSecretView   Permission = "secret_view"
	PermissionSecretEdit   Permission = "secret_edit"
	PermissionSecretDelete Permission = "secret_delete"
	PermissionSecretAccess Permission = "secret_access"
)

const (
	/*
		----- COMMIT CHECK -----
	*/
	PermissionCommitCheckReport Permission = "commitCheck_report"
)
