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

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrResourceNotFound",
			err:  ErrResourceNotFound,
			msg:  "resource not found",
		},
		{
			name: "ErrDuplicate",
			err:  ErrDuplicate,
			msg:  "resource is a duplicate",
		},
		{
			name: "ErrForeignKeyViolation",
			err:  ErrForeignKeyViolation,
			msg:  "foreign resource does not exists",
		},
		{
			name: "ErrVersionConflict",
			err:  ErrVersionConflict,
			msg:  "resource version conflict",
		},
		{
			name: "ErrPathTooLong",
			err:  ErrPathTooLong,
			msg:  "the path is too long",
		},
		{
			name: "ErrPrimaryPathAlreadyExists",
			err:  ErrPrimaryPathAlreadyExists,
			msg:  "primary path already exists for resource",
		},
		{
			name: "ErrPrimaryPathRequired",
			err:  ErrPrimaryPathRequired,
			msg:  "path has to be primary",
		},
		{
			name: "ErrAliasPathRequired",
			err:  ErrAliasPathRequired,
			msg:  "path has to be an alias",
		},
		{
			name: "ErrPrimaryPathCantBeDeleted",
			err:  ErrPrimaryPathCantBeDeleted,
			msg:  "primary path can't be deleted",
		},
		{
			name: "ErrNoChangeInRequestedMove",
			err:  ErrNoChangeInRequestedMove,
			msg:  "the requested move doesn't change anything",
		},
		{
			name: "ErrIllegalMoveCyclicHierarchy",
			err:  ErrIllegalMoveCyclicHierarchy,
			msg:  "the requested move is not permitted as it would cause a cyclic dependency",
		},
		{
			name: "ErrSpaceWithChildsCantBeDeleted",
			err:  ErrSpaceWithChildsCantBeDeleted,
			msg:  "the space can't be deleted as it still contains spaces or repos",
		},
		{
			name: "ErrPreConditionFailed",
			err:  ErrPreConditionFailed,
			msg:  "precondition failed",
		},
		{
			name: "ErrLicenseNotFound",
			err:  ErrLicenseNotFound,
			msg:  "license not found",
		},
		{
			name: "ErrLicenseExpired",
			err:  ErrLicenseExpired,
			msg:  "license expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err, "error should not be nil")
			assert.Equal(t, tt.msg, tt.err.Error(), "error message should match")
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	// Verify that all errors are distinct
	allErrors := []error{
		ErrResourceNotFound,
		ErrDuplicate,
		ErrForeignKeyViolation,
		ErrVersionConflict,
		ErrPathTooLong,
		ErrPrimaryPathAlreadyExists,
		ErrPrimaryPathRequired,
		ErrAliasPathRequired,
		ErrPrimaryPathCantBeDeleted,
		ErrNoChangeInRequestedMove,
		ErrIllegalMoveCyclicHierarchy,
		ErrSpaceWithChildsCantBeDeleted,
		ErrPreConditionFailed,
		ErrLicenseNotFound,
		ErrLicenseExpired,
	}

	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j {
				assert.False(t, errors.Is(err1, err2), "errors should be distinct: %v vs %v", err1, err2)
			}
		}
	}
}

func TestErrorsCanBeCompared(t *testing.T) {
	// Test that errors can be compared using errors.Is
	err := ErrResourceNotFound
	assert.True(t, errors.Is(err, ErrResourceNotFound), "should match ErrResourceNotFound")
	assert.False(t, errors.Is(err, ErrDuplicate), "should not match ErrDuplicate")
}
