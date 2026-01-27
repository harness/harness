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
	"net/url"
	"regexp"
	"regexp/syntax"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// Placeholder that must appear in target URL for prefix_num type.
	PlaceholderNum = "<num>"
	// Placeholder that must appear in target URL for prefix_alphanumeric type.
	PlaceholderAlphanum = "<alphanum>"
)

type AutoLinkInput struct {
	Type      enum.AutoLinkType `json:"type"`
	Pattern   string            `json:"pattern"`
	TargetURL string            `json:"target_url"`
}

func (in *AutoLinkInput) Sanitize() error {
	if in.Pattern == "" {
		return usererror.BadRequest("pattern is required")
	}

	if in.Type == enum.AutoLinkTypePrefixWithNumValue ||
		in.Type == enum.AutoLinkTypePrefixWithAlphanumericValue {
		if strings.Contains(in.Pattern, " ") {
			return usererror.BadRequest("pattern cannot contain spaces")
		}
	}

	if in.Type == enum.AutoLinkTypeRegex {
		// check for RE2 compliance ref: https://github.com/google/re2/wiki/WhyRE2
		_, err := syntax.Parse(in.Pattern, syntax.Perl)
		if err != nil {
			return usererror.BadRequestf("invalid regex pattern: %s", err)
		}
	}

	if in.TargetURL == "" {
		return usererror.BadRequest("target URL is required")
	}

	const maxURLLength = 2000 // Common URL length limit
	if len(in.TargetURL) > maxURLLength {
		return usererror.BadRequestf("target URL cannot exceed %d characters", maxURLLength)
	}

	parsedURL, err := url.Parse(in.TargetURL)
	if err != nil {
		return usererror.BadRequestf("invalid target URL: %s", err)
	}

	if parsedURL.Hostname() == "" {
		return usererror.BadRequest("target URL must have a non-empty host")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return usererror.BadRequest("target URL scheme must be either http or https")
	}

	// For prefix types, validate that target URL contains the pattern followed by the correct placeholder.
	if err := validatePrefixURL(in.Type, in.Pattern, in.TargetURL); err != nil {
		return err
	}

	return nil
}

// validatePrefixURL validates that for prefix types, the target URL contains
// the pattern followed by the appropriate placeholder (<num> or <alphanum>).
func validatePrefixURL(linkType enum.AutoLinkType, pattern, targetURL string) error {
	//nolint:exhaustive
	switch linkType {
	case enum.AutoLinkTypePrefixWithNumValue:
		expectedInURL := pattern + PlaceholderNum
		if !strings.Contains(targetURL, expectedInURL) {
			return usererror.BadRequestf(
				"target URL must contain '%s' for prefix_num type", expectedInURL)
		}
	case enum.AutoLinkTypePrefixWithAlphanumericValue:
		expectedInURL := pattern + PlaceholderAlphanum
		if !strings.Contains(targetURL, expectedInURL) {
			return usererror.BadRequestf(
				"target URL must contain '%s' for prefix_alphanumeric type", expectedInURL)
		}
	}
	return nil
}

// validateURL validates the target URL format.
func validateURL(targetURL string) error {
	const maxURLLength = 2000
	if len(targetURL) > maxURLLength {
		return usererror.BadRequestf("target URL cannot exceed %d characters", maxURLLength)
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return usererror.BadRequestf("invalid target URL: %s", err)
	}

	if parsedURL.Hostname() == "" {
		return usererror.BadRequest("target URL must have a non-empty host")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return usererror.BadRequest("target URL scheme must be either http or https")
	}

	return nil
}

// ConvertPatternToRegex converts a user-friendly pattern to regex format based on type.
// For prefix_num: "JIRA-" becomes "JIRA-([0-9]+)"
// For prefix_alphanumeric: "REF-" becomes "REF-([a-zA-Z0-9]+)"
// For regex: pattern is returned as-is.
func ConvertPatternToRegex(linkType enum.AutoLinkType, pattern string) string {
	//nolint:exhaustive
	switch linkType {
	case enum.AutoLinkTypePrefixWithNumValue:
		return regexp.QuoteMeta(pattern) + "([0-9]+)"
	case enum.AutoLinkTypePrefixWithAlphanumericValue:
		return regexp.QuoteMeta(pattern) + "([a-zA-Z0-9]+)"
	default:
		return pattern
	}
}

func (s *Service) Create(
	ctx context.Context,
	principalID int64,
	spaceID, repoID *int64,
	in *AutoLinkInput,
) (*types.AutoLink, error) {
	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	regexPattern := ConvertPatternToRegex(in.Type, in.Pattern)

	autolink := &types.AutoLink{
		SpaceID:   spaceID,
		RepoID:    repoID,
		Type:      in.Type,
		Pattern:   regexPattern,
		TargetURL: in.TargetURL,
		CreatedBy: principalID,
		UpdatedBy: principalID,
		Created:   now,
		Updated:   now,
	}

	err := s.autoLinkStore.Create(ctx, autolink)
	if err != nil {
		return nil, fmt.Errorf("failed to create autolink: %w", err)
	}

	return autolink, nil
}
