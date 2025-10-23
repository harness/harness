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

package limiter

import (
	"context"

	"github.com/harness/gitness/types/enum"
)

// Gitspace is an interface for managing gitspace limitations.
type Gitspace interface {
	// Usage checks if the total usage for the root space and all sub-spaces is under a limit.
	Usage(ctx context.Context, spaceID int64, infraProviderType enum.InfraProviderType) error
}

var _ Gitspace = (*UnlimitedUsage)(nil)

type UnlimitedUsage struct {
}

// NewUnlimitedUsage creates a new instance of UnlimitedGitspace.
func NewUnlimitedUsage() Gitspace {
	return UnlimitedUsage{}
}

func (UnlimitedUsage) Usage(_ context.Context, _ int64, _ enum.InfraProviderType) error {
	return nil
}
