// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package store

import "errors"

var (
	ErrResourceNotFound             = errors.New("Resource not found")
	ErrDuplicate                    = errors.New("Resource is a duplicate")
	ErrPathTooLong                  = errors.New("The path is too long")
	ErrPrimaryPathAlreadyExists     = errors.New("Primary path already exists for resource.")
	ErrPrimaryPathRequired          = errors.New("Path has to be primary.")
	ErrAliasPathRequired            = errors.New("Path has to be an alias.")
	ErrPrimaryPathCantBeDeleted     = errors.New("Primary path can't be deleted.")
	ErrNoChangeInRequestedMove      = errors.New(("The requested move doesn't change anything."))
	ErrIllegalMoveCyclicHierarchy   = errors.New(("The requested move is not permitted as it would cause a cyclic depdency."))
	ErrSpaceWithChildsCantBeDeleted = errors.New("The space can't be deleted as it still contains spaces or repos.")
)
