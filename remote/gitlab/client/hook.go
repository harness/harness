// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"encoding/json"
	"fmt"
)

// ParseHook parses hook payload from GitLab
func ParseHook(payload []byte) (*HookPayload, error) {
	hp := HookPayload{}
	if err := json.Unmarshal(payload, &hp); err != nil {
		return nil, err
	}

	// Basic sanity check
	switch {
	case len(hp.ObjectKind) == 0:
		// Assume this is a post-receive within repository
		if len(hp.After) == 0 {
			return nil, fmt.Errorf("Invalid hook received, commit hash not found.")
		}
	case hp.ObjectKind == "push":
		if hp.Repository == nil {
			return nil, fmt.Errorf("Invalid push hook received, attributes not found")
		}
	case hp.ObjectKind == "tag_push":
		if hp.Repository == nil {
			return nil, fmt.Errorf("Invalid tag push hook received, attributes not found")
		}
	case hp.ObjectKind == "issue":
		fallthrough
	case hp.ObjectKind == "merge_request":
		if hp.ObjectAttributes == nil {
			return nil, fmt.Errorf("Invalid hook received, attributes not found.")
		}
	default:
		return nil, fmt.Errorf("Invalid hook received, payload format not recognized.")
	}

	return &hp, nil
}
