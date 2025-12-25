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

package settings

import (
	"context"
	"fmt"
)

// RepoGet is a helper method for getting a setting of a specific type for a repo.
func RepoGet[T any](
	ctx context.Context,
	s *Service,
	repoID int64,
	key Key,
	dflt T,
) (T, error) {
	var out T
	ok, err := s.RepoGet(ctx, repoID, key, &out)
	if err != nil {
		return out, err
	}

	if !ok {
		return dflt, nil
	}

	return out, nil
}

// RepoGetRequired is a helper method for getting a setting of a specific type for a repo.
// If the setting isn't found, an error is returned.
func RepoGetRequired[T any](
	ctx context.Context,
	s *Service,
	repoID int64,
	key Key,
) (T, error) {
	var out T
	ok, err := s.RepoGet(ctx, repoID, key, &out)
	if err != nil {
		return out, err
	}

	if !ok {
		return out, fmt.Errorf("setting %q not found", key)
	}

	return out, nil
}

// SystemGet is a helper method for getting a setting of a specific type for the system.
func SystemGet[T any](
	ctx context.Context,
	s *Service,
	key Key,
	dflt T,
) (T, error) {
	var out T
	ok, err := s.SystemGet(ctx, key, &out)
	if err != nil {
		return out, err
	}

	if !ok {
		return dflt, nil
	}

	return out, nil
}

// SystemGetRequired is a helper method for getting a setting of a specific type for the system.
// If the setting isn't found, an error is returned.
func SystemGetRequired[T any](
	ctx context.Context,
	s *Service,
	key Key,
) (T, error) {
	var out T
	ok, err := s.SystemGet(ctx, key, &out)
	if err != nil {
		return out, err
	}

	if !ok {
		return out, fmt.Errorf("setting %q not found", key)
	}

	return out, nil
}
