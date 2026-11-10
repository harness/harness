// Copyright 2026 Harness, Inc.
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

package codeowners

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
)

// ContentValidator validates CODEOWNERS content. It is registered for the configured CODEOWNERS paths.
type ContentValidator struct {
	service *Service
}

// NewContentValidator creates a content validator backed by the CODEOWNERS service.
func NewContentValidator(service *Service) *ContentValidator {
	return &ContentValidator{service: service}
}

// Paths returns the configured CODEOWNERS file paths.
func (v *ContentValidator) Paths() []string {
	return v.service.config.FilePaths
}

// Validate validates raw CODEOWNERS content.
func (v *ContentValidator) Validate(
	ctx context.Context,
	_ *types.RepositoryCore,
	content []byte,
) ([]types.ContentViolation, error) {
	validation, err := v.service.ValidateContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to validate CODEOWNERS content: %w", err)
	}

	violations := make([]types.ContentViolation, 0, len(validation.Violations))
	for _, violation := range validation.Violations {
		violations = append(violations, types.ContentViolation{
			Code:    string(violation.Code),
			Message: violation.Message,
			Params:  violation.Params,
			Line:    violation.Line,
		})
	}
	return violations, nil
}
