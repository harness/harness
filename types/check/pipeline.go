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
	// ErrPipelineIdentifier is returned when the pipeline
	// slug is an invalid format.
	ErrPipelineIdentifier = errors.New("Invalid pipeline identifier")

	// ErrPipelineIdentifierLen is returned when the pipeline
	// name exceeds the maximum number of characters.
	ErrPipelineIdentifierLen = errors.New("Pipeline identifier cannot exceed 250 characters")

	// ErrPipelineNameLen is returned when the pipeline name
	// exceeds the maximum number of characters.
	ErrPipelineNameLen = errors.New("Pipeline name cannot exceed 250 characters")

	// ErrPipelineDescLen is returned when the pipeline desc
	// exceeds the maximum number of characters.
	ErrPipelineDescLen = errors.New("Pipeline description cannot exceed 250 characters")
)

// Pipeline returns true if the Pipeline if valid.
func Pipeline(pipeline *types.Pipeline) (bool, error) {
	if !slug.IsSlug(pipeline.Slug) {
		return false, ErrPipelineIdentifier
	}
	if len(pipeline.Slug) > 250 {
		return false, ErrPipelineIdentifierLen
	}
	if len(pipeline.Name) > 250 {
		return false, ErrPipelineNameLen
	}
	if len(pipeline.Desc) > 500 {
		return false, ErrPipelineDescLen
	}
	return true, nil
}
