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

// Package checkreq lets an external owner of status checks declare which of its
// checks must pass before a pull request can be merged, without expressing that
// requirement as a branch protection rule.
//
// Requirements resolved here are deliberately kept outside the protection
// package: protection violations can be waived by a rule's bypass list, whereas
// a non-bypassable requirement must hold for everybody. The provider owns the
// storage of its own configuration; gitness only ever consumes check
// identifiers.
package checkreq

import (
	"context"

	"github.com/harness/gitness/types"
)

// Requirements is the set of checks a provider requires for a single pull request.
type Requirements struct {
	// Required checks must have reported success. An explicit bypass on the
	// check (Check.BypassedBy) does not satisfy them.
	Required []string
	// Bypassable checks must have reported success, or carry an explicit
	// bypass on the check (Check.BypassedBy).
	Bypassable []string
}

func (r Requirements) IsEmpty() bool {
	return len(r.Required) == 0 && len(r.Bypassable) == 0
}

// Provider resolves the checks required for a pull request from its own storage.
type Provider interface {
	// RequiredChecks returns the checks required for the given pull request.
	RequiredChecks(
		ctx context.Context,
		repo *types.RepositoryCore,
		pr *types.PullReq,
	) (Requirements, error)
}

// NoopProvider is the default provider. It keeps standalone gitness free of any
// external check requirements.
type NoopProvider struct{}

func (NoopProvider) RequiredChecks(
	context.Context,
	*types.RepositoryCore,
	*types.PullReq,
) (Requirements, error) {
	return Requirements{}, nil
}
