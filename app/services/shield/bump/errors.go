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

// Package bump implements the deterministic_bump fix strategy of Shield: it raises a vulnerable
// dependency to the minimum fixed version, updates the manifest (and go.sum) and opens a pull
// request, without using any LLM quota.
//
// Limits: transitive changes are never resolved (no `go mod tidy`, no lockfile regeneration), so CI
// must validate every pull request. Whenever the fix cannot be applied mechanically a NeedsAgentError
// is returned and the finding has to be handled by the fix agent.
package bump

import (
	"errors"
	"fmt"
)

// ErrUnsupported is returned for ecosystems or files that the deterministic bump does not handle.
var ErrUnsupported = errors.New("bump: unsupported manifest")

// NeedsAgentError reports that the fix cannot be applied deterministically and needs the agent.
type NeedsAgentError struct {
	Reason string
}

func (e *NeedsAgentError) Error() string { return "bump: needs agent: " + e.Reason }

func needsAgent(format string, args ...any) error {
	return &NeedsAgentError{Reason: fmt.Sprintf(format, args...)}
}

// IsNeedsAgent tells whether err is (or wraps) a NeedsAgentError.
func IsNeedsAgent(err error) bool {
	var n *NeedsAgentError
	return errors.As(err, &n)
}
