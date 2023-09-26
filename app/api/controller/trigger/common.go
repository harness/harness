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

package trigger

import (
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

const (
	// triggerMaxSecretLength defines the max allowed length of a trigger secret.
	// TODO: Check whether this is sufficient for other SCM providers once we
	// add support. For now it's good to have a limit and increase if needed.
	triggerMaxSecretLength = 4096
)

// checkSecret validates the secret of a trigger.
func checkSecret(secret string) error {
	if len(secret) > triggerMaxSecretLength {
		return check.NewValidationErrorf("The secret of a trigger can be at most %d characters long.",
			triggerMaxSecretLength)
	}

	return nil
}

// checkActions validates the trigger actions.
func checkActions(actions []enum.TriggerAction) error {
	// ignore duplicates here, should be deduplicated later
	for _, action := range actions {
		if _, ok := action.Sanitize(); !ok {
			return check.NewValidationErrorf("The provided trigger action '%s' is invalid.", action)
		}
	}

	return nil
}

// deduplicateActions de-duplicates the actions provided by in the trigger.
func deduplicateActions(in []enum.TriggerAction) []enum.TriggerAction {
	if len(in) == 0 {
		return []enum.TriggerAction{}
	}

	actionSet := make(map[enum.TriggerAction]struct{})
	out := make([]enum.TriggerAction, 0, len(in))
	for _, action := range in {
		if _, ok := actionSet[action]; ok {
			continue
		}
		actionSet[action] = struct{}{}
		out = append(out, action)
	}

	return out
}
