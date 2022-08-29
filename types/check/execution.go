// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"

	"github.com/harness/gitness/types"

	"github.com/gosimple/slug"
)

var (
	// ErrExecutionIdentifier is returned when the execution
	// slug is an invalid format.
	ErrExecutionIdentifier = errors.New("Invalid execution identifier")

	// ErrExecutionIdentifierLen is returned when the execution
	// name exceeds the maximum number of characters.
	ErrExecutionIdentifierLen = errors.New("Execution identifier cannot exceed 250 characters")

	// ErrExecutionNameLen is returned when the execution name
	// exceeds the maximum number of characters.
	ErrExecutionNameLen = errors.New("Execution name cannot exceed 250 characters")

	// ErrExecutionDescLen is returned when the execution desc
	// exceeds the maximum number of characters.
	ErrExecutionDescLen = errors.New("Execution description cannot exceed 250 characters")
)

// Execution returns true if the Execution if valid.
func Execution(execution *types.Execution) (bool, error) {
	if !slug.IsSlug(execution.Slug) {
		return false, ErrExecutionIdentifier
	}
	if len(execution.Slug) > 250 {
		return false, ErrExecutionIdentifierLen
	}
	if len(execution.Name) > 250 {
		return false, ErrExecutionNameLen
	}
	if len(execution.Desc) > 500 {
		return false, ErrExecutionDescLen
	}
	return true, nil
}
