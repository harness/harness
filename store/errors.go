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

package store

import "errors"

var (
	ErrResourceNotFound           = errors.New("resource not found")
	ErrDuplicate                  = errors.New("resource is a duplicate")
	ErrForeignKeyViolation        = errors.New("foreign resource does not exists")
	ErrVersionConflict            = errors.New("resource version conflict")
	ErrPathTooLong                = errors.New("the path is too long")
	ErrPrimaryPathAlreadyExists   = errors.New("primary path already exists for resource")
	ErrPrimaryPathRequired        = errors.New("path has to be primary")
	ErrAliasPathRequired          = errors.New("path has to be an alias")
	ErrPrimaryPathCantBeDeleted   = errors.New("primary path can't be deleted")
	ErrNoChangeInRequestedMove    = errors.New("the requested move doesn't change anything")
	ErrIllegalMoveCyclicHierarchy = errors.New("the requested move is not permitted as it would cause a " +
		"cyclic depdency")
	ErrSpaceWithChildsCantBeDeleted = errors.New("the space can't be deleted as it still contains " +
		"spaces or repos")
	ErrPreConditionFailed = errors.New("precondition failed")
)
