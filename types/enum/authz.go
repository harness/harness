// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ResourceType represents the different types of resources that can be guarded with permissions.
type ResourceType string

const (
	ResourceTypeSpace          ResourceType = "SPACE"
	ResourceTypeRepo           ResourceType = "REPOSITORY"
	ResourceTypeServiceAccount ResourceType = "SERVICEACCOUNT"
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
	PermissionRepoCreate Permission = "repository_create"
	PermissionRepoView   Permission = "repository_view"
	PermissionRepoEdit   Permission = "repository_edit"
	PermissionRepoDelete Permission = "repository_delete"
)

const (
	/*
		----- REPOSITORY -----
	*/
	PermissionServiceAccountCreate Permission = "serviceaccount_create"
	PermissionServiceAccountView   Permission = "serviceaccount_view"
	PermissionServiceAccountEdit   Permission = "serviceaccount_edit"
	PermissionServiceAccountDelete Permission = "serviceaccount_delete"
)

// AccessGrant represents the access grants a token or sshkey can have.
// Keep as int64 to allow for simpler+faster lookup of grants for a given token
// as we don't have to store an array field or need to do a join / 2nd db call.
// Multiple grants can be combined using the bit-wise or operation.
// ASSUMPTION: we don't need more than 63 grants!
//
// NOTE: A grant is always restricted by the principal permissions
//
// TODO: Beter name, access grant and permission might be to close in terminology?
type AccessGrant int64

const (
	// no grants - useless token.
	AccessGrantNone AccessGrant = 0

	// privacy related grants.
	AccessGrantPublic  AccessGrant = 1 << 0 // 1
	AccessGrantPrivate AccessGrant = 1 << 1 // 2

	// api related grants (spaces / repos, ...).
	AccessGrantAPICreate AccessGrant = 1 << 10 // 1024
	AccessGrantAPIView   AccessGrant = 1 << 11 // 2048
	AccessGrantAPIEdit   AccessGrant = 1 << 12 // 4096
	AccessGrantAPIDelete AccessGrant = 1 << 13 // 8192

	// code related grants.
	AccessGrantCodeRead  AccessGrant = 1 << 20 // 1048576
	AccessGrantCodeWrite AccessGrant = 1 << 21 // 2097152

	// grants everything - for user sessions.
	AccessGrantAll AccessGrant = 1<<63 - 1
)
