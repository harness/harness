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

package autolink

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type AutoLinkUpdateInput struct {
	Type      *enum.AutoLinkType `json:"type,omitempty"`
	Pattern   *string            `json:"pattern,omitempty"`
	TargetURL *string            `json:"target_url,omitempty"`
}

func (s *Service) Update(
	ctx context.Context,
	autolinkID int64,
	principalID int64,
	in *AutoLinkUpdateInput,
) (*types.AutoLink, error) {
	autolink, err := s.autoLinkStore.Find(ctx, autolinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get autolink: %w", err)
	}

	// Determine the effective values after update.
	effectiveType := autolink.Type
	if in.Type != nil {
		effectiveType = *in.Type
	}

	effectivePattern := autolink.Pattern
	if in.Pattern != nil && *in.Pattern != "" {
		effectivePattern = *in.Pattern
	}

	effectiveURL := autolink.TargetURL
	if in.TargetURL != nil && *in.TargetURL != "" {
		effectiveURL = *in.TargetURL
	}

	if in.TargetURL != nil && *in.TargetURL != "" {
		if err := validateURL(*in.TargetURL); err != nil {
			return nil, err
		}
	}

	userPattern := effectivePattern
	if in.Pattern != nil && *in.Pattern != "" {
		userPattern = *in.Pattern
	} else if isPrefixType(effectiveType) {
		if in.Type != nil && *in.Type != autolink.Type {
			return nil, fmt.Errorf("pattern must be provided when changing autolink type")
		}
		userPattern = ""
	}

	if userPattern != "" && isPrefixType(effectiveType) {
		if err := validatePrefixURL(effectiveType, userPattern, effectiveURL); err != nil {
			return nil, err
		}
	}

	if in.Type != nil {
		autolink.Type = *in.Type
	}

	if in.Pattern != nil && *in.Pattern != "" {
		autolink.Pattern = ConvertPatternToRegex(effectiveType, *in.Pattern)
	}

	if in.TargetURL != nil && *in.TargetURL != "" {
		autolink.TargetURL = *in.TargetURL
	}

	autolink.UpdatedBy = principalID

	err = s.autoLinkStore.Update(ctx, autolink)
	if err != nil {
		return nil, fmt.Errorf("failed to update autolink: %w", err)
	}

	return autolink, nil
}

func isPrefixType(t enum.AutoLinkType) bool {
	return t == enum.AutoLinkTypePrefixWithNumValue ||
		t == enum.AutoLinkTypePrefixWithAlphanumericValue
}
