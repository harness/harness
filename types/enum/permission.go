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
	ResourceTypeConnector      ResourceType = "CONNECTOR"
	ResourceTypeTemplate       ResourceType = "TEMPLATE"
	ResourceTypeGitspace       ResourceType = "GITSPACE"
	ResourceTypeInfraProvider  ResourceType = "INFRAPROVIDER"
)

// Permission represents the different types of permissions a principal can have.
type Permission string

const (
	/*
	   ----- SPACE -----
	*/
	PermissionSpaceView   Permission = "space_view"
	PermissionSpaceEdit   Permission = "space_edit"
	PermissionSpaceDelete Permission = "space_delete"
)

const (
	/*
		----- REPOSITORY -----
	*/
	PermissionRepoView              Permission = "repo_view"
	PermissionRepoEdit              Permission = "repo_edit"
	PermissionRepoDelete            Permission = "repo_delete"
	PermissionRepoPush              Permission = "repo_push"
	PermissionRepoReview            Permission = "repo_review"
	PermissionRepoReportCommitCheck Permission = "repo_reportCommitCheck"
)

const (
	/*
		----- USER -----
	*/
	PermissionUserView      Permission = "user_view"
	PermissionUserEdit      Permission = "user_edit"
	PermissionUserDelete    Permission = "user_delete"
	PermissionUserEditAdmin Permission = "user_editAdmin"
)

const (
	/*
		----- SERVICE ACCOUNT -----
	*/
	PermissionServiceAccountView   Permission = "serviceaccount_view"
	PermissionServiceAccountEdit   Permission = "serviceaccount_edit"
	PermissionServiceAccountDelete Permission = "serviceaccount_delete"
)

const (
	/*
		----- SERVICE -----
	*/
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
		----- CONNECTOR -----
	*/
	PermissionConnectorView   Permission = "connector_view"
	PermissionConnectorEdit   Permission = "connector_edit"
	PermissionConnectorDelete Permission = "connector_delete"
	PermissionConnectorAccess Permission = "connector_access"
)

const (
	/*
		----- TEMPLATE -----
	*/
	PermissionTemplateView   Permission = "template_view"
	PermissionTemplateEdit   Permission = "template_edit"
	PermissionTemplateDelete Permission = "template_delete"
	PermissionTemplateAccess Permission = "template_access"
)

const (
	/*
		----- GITSPACE -----
	*/
	PermissionGitspaceView   Permission = "gitspace_view"
	PermissionGitspaceEdit   Permission = "gitspace_edit"
	PermissionGitspaceDelete Permission = "gitspace_delete"
	PermissionGitspaceAccess Permission = "gitspace_access"
)

const (
	/*
		----- INFRAPROVIDER -----
	*/
	PermissionInfraProviderView   Permission = "infraprovider_view"
	PermissionInfraProviderEdit   Permission = "infraprovider_edit"
	PermissionInfraProviderDelete Permission = "infraprovider_delete"
	PermissionInfraProviderAccess Permission = "infraprovider_access"
)
