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
	"fmt"

	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

var (
	// KeyAIMode [enum.AIMode] controls what data may be sent to AI models for a repo.
	// An empty value means "use the default for the repo's visibility".
	KeyAIMode     Key = "ai_mode"
	DefaultAIMode     = enum.AIMode("")
)

// DefaultAIModeForVisibility returns the AI mode used when no explicit mode is configured:
// public repos default to full, private repos to decisions-only.
func DefaultAIModeForVisibility(isPublic bool) enum.AIMode {
	if isPublic {
		return enum.AIModeFull
	}
	return enum.AIModeDecisionsOnly
}

// ValidateAIMode validates a user provided AI mode. An empty value is valid (reset to default).
func ValidateAIMode(mode *enum.AIMode) error {
	if mode == nil || *mode == "" {
		return nil
	}
	if _, ok := mode.Sanitize(); !ok {
		return fmt.Errorf("invalid ai mode %q, allowed values: %v", *mode, enum.AIModes)
	}
	return nil
}

func defaultAIModePtr() *enum.AIMode {
	return ptr.Of(DefaultAIMode)
}
