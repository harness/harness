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

package enum

import (
	"encoding/json"
	"fmt"
)

type AIAgent string

func (a AIAgent) Enum() []interface{} {
	return toInterfaceSlice(aiAgentTypes)
}

func (a AIAgent) String() string {
	return string(a)
}

var aiAgentTypes = []AIAgent{
	AIAgentClaudeCode,
}

const (
	AIAgentClaudeCode AIAgent = "claude-code"
)

func (a *AIAgent) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "" {
		*a = ""
		return nil
	}

	for _, validType := range aiAgentTypes {
		if AIAgent(str) == validType {
			*a = validType
			return nil
		}
	}

	return fmt.Errorf("invalid AI agent type: %s", str)
}
