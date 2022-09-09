// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package errs

import "errors"

var (
	Internal                        = errors.New("Internal error occured - Please contact operator for more information.")
	NotAuthenticated                = errors.New("Not authenticated.")
	NotAuthorized                   = errors.New("Not authorized.")
	RepositoryRequired              = errors.New("The operation requires a repository.")
	PathEmpty                       = errors.New("Path is empty.")
	PrimaryPathAlreadyExists        = errors.New("Primary path already exists for resource.")
	AliasPathRequired               = errors.New("Path has to be an alias.")
	PrimaryPathRequired             = errors.New("Path has to be primary.")
	PrimaryPathCantBeDeleted        = errors.New("Primary path can't be deleted.")
	NoChangeInRequestedMove         = errors.New(("The requested move doesn't change anything."))
	IllegalMoveCyclicHierarchy      = errors.New(("The requested move is not permitted as it would cause a cyclic depdency."))
	SpaceWithChildsCantBeDeleted    = errors.New("The space can't be deleted as it still contains spaces or repos.")
	RepoReferenceNotFoundInRequest  = errors.New("No repository reference found in request.")
	SpaceReferenceNotFoundInRequest = errors.New("No space reference found in request.")
	NoPermissionCheckProvided       = errors.New("No permission checks provided")
)
