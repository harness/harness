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

package insights

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

const (
	// DefaultRateLimitPerMinute is the event rate limit of new projects.
	DefaultRateLimitPerMinute = 600
	// MaxRateLimitPerMinute is the highest configurable rate limit.
	MaxRateLimitPerMinute = 100000
)

var projectIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

// CreateProject creates a project with a fresh public key in the repository.
func (s *Service) CreateProject(
	ctx context.Context,
	repoID, createdBy int64,
	in *types.InsightsProjectCreateInput,
) (*types.InsightsProject, error) {
	if !projectIdentifierRegex.MatchString(in.Identifier) {
		return nil, usererror.BadRequest(
			"Project identifier must start with a letter or digit, contain only letters, digits, " +
				"'.', '_' and '-', and have at most 64 characters.")
	}

	rate := DefaultRateLimitPerMinute
	if in.RateLimitPerMinute != nil {
		rate = *in.RateLimitPerMinute
		if rate < 0 || rate > MaxRateLimitPerMinute {
			return nil, usererror.BadRequestf("rate_limit_per_minute must be between 0 and %d.", MaxRateLimitPerMinute)
		}
	}

	sample := 1.0
	if in.SampleRate != nil {
		sample = *in.SampleRate
		if sample < 0 || sample > 1 {
			return nil, usererror.BadRequest("sample_rate must be between 0 and 1.")
		}
	}

	key, err := newPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	now := time.Now().UnixMilli()
	p := &types.InsightsProject{
		RepoID:             repoID,
		Identifier:         in.Identifier,
		PublicKey:          key,
		Enabled:            true,
		RateLimitPerMinute: rate,
		SampleRate:         sample,
		CreatedBy:          createdBy,
		Created:            now,
		Updated:            now,
	}

	if err = s.projects.Create(ctx, p); err != nil {
		if errors.Is(err, gitness_store.ErrDuplicate) {
			return nil, usererror.ConflictWithPayload("An Insights project with this identifier already exists.")
		}
		return nil, fmt.Errorf("failed to create insights project: %w", err)
	}

	return p, nil
}

// ListProjects returns the projects of the repository.
func (s *Service) ListProjects(ctx context.Context, repoID int64) ([]*types.InsightsProject, error) {
	list, err := s.projects.List(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to list insights projects: %w", err)
	}
	return list, nil
}

func newPublicKey() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
