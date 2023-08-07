// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
))

var membershipRoleExecutorPermissions = slices.Clip(slices.Insert(membershipRoleReaderPermissions, 0,
	PermissionCommitCheckReport,
	PermissionPipelineExecute,
))

var membershipRoleContributorPermissions = slices.Clip(slices.Insert(membershipRoleReaderPermissions, 0,
	PermissionRepoPush,
))

var membershipRoleSpaceOwnerPermissions = slices.Clip(slices.Insert(membershipRoleReaderPermissions, 0,
	PermissionRepoEdit,
	PermissionRepoDelete,
	PermissionRepoPush,
	PermissionCommitCheckReport,

	PermissionSpaceEdit,
	PermissionSpaceCreate,
	PermissionSpaceDelete,

	PermissionServiceAccountCreate,
	PermissionServiceAccountEdit,
	PermissionServiceAccountDelete,

	PermissionPipelineEdit,
	PermissionPipelineExecute,
	PermissionPipelineDelete,
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
