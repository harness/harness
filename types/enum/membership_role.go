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

import "golang.org/x/exp/slices"

// MembershipRole represents the different level of space memberships (permission set).
type MembershipRole string

func (MembershipRole) Enum() []interface{}                      { return toInterfaceSlice(MembershipRoles) }
func (m MembershipRole) Sanitize() (MembershipRole, bool)       { return Sanitize(m, GetAllMembershipRoles) }
func GetAllMembershipRoles() ([]MembershipRole, MembershipRole) { return MembershipRoles, "" }

var MembershipRoles = sortEnum([]MembershipRole{
	MembershipRoleReader,
	MembershipRoleExecutor,
	MembershipRoleContributor,
	MembershipRoleSpaceOwner,
})

var membershipRoleReaderPermissions = slices.Clip(slices.Insert([]Permission{}, 0,
	PermissionRepoView,
	PermissionSpaceView,
	PermissionServiceAccountView,
	PermissionPipelineView,
	PermissionSecretView,
	PermissionConnectorView,
	PermissionTemplateView,
	PermissionGitspaceView,
	PermissionInfraProviderView,
))

var membershipRoleExecutorPermissions = slices.Clip(slices.Insert(membershipRoleReaderPermissions, 0,
	PermissionRepoReportCommitCheck,
	PermissionPipelineExecute,
	PermissionSecretAccess,
	PermissionConnectorAccess,
	PermissionTemplateAccess,
	PermissionGitspaceAccess,
	PermissionInfraProviderAccess,
))

var membershipRoleContributorPermissions = slices.Clip(slices.Insert(membershipRoleReaderPermissions, 0,
	PermissionRepoPush,
	PermissionRepoReview,
))

var membershipRoleSpaceOwnerPermissions = slices.Clip(slices.Insert(membershipRoleReaderPermissions, 0,
	PermissionRepoEdit,
	PermissionRepoDelete,
	PermissionRepoPush,
	PermissionRepoReportCommitCheck,
	PermissionRepoReview,

	PermissionSpaceEdit,
	PermissionSpaceDelete,

	PermissionServiceAccountEdit,
	PermissionServiceAccountDelete,

	PermissionPipelineEdit,
	PermissionPipelineExecute,
	PermissionPipelineDelete,

	PermissionSecretAccess,
	PermissionSecretDelete,
	PermissionSecretEdit,

	PermissionConnectorAccess,
	PermissionConnectorDelete,
	PermissionConnectorEdit,

	PermissionTemplateAccess,
	PermissionTemplateDelete,
	PermissionTemplateEdit,

	PermissionGitspaceEdit,
	PermissionGitspaceDelete,
	PermissionGitspaceAccess,

	PermissionInfraProviderEdit,
	PermissionInfraProviderDelete,
	PermissionInfraProviderAccess,
))

func init() {
	slices.Sort(membershipRoleReaderPermissions)
	slices.Sort(membershipRoleExecutorPermissions)
	slices.Sort(membershipRoleContributorPermissions)
	slices.Sort(membershipRoleSpaceOwnerPermissions)
}

// Permissions returns the list of permissions for the role.
func (m MembershipRole) Permissions() []Permission {
	switch m {
	case MembershipRoleReader:
		return membershipRoleReaderPermissions
	case MembershipRoleExecutor:
		return membershipRoleExecutorPermissions
	case MembershipRoleContributor:
		return membershipRoleContributorPermissions
	case MembershipRoleSpaceOwner:
		return membershipRoleSpaceOwnerPermissions
	default:
		return nil
	}
}

const (
	MembershipRoleReader      MembershipRole = "reader"
	MembershipRoleExecutor    MembershipRole = "executor"
	MembershipRoleContributor MembershipRole = "contributor"
	MembershipRoleSpaceOwner  MembershipRole = "space_owner"
)
