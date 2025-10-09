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

import "fmt"

// ResolverType represents the type of resolver.
type ResolverType string

const (
	// ResolverTypeStep is a step level resolver.
	ResolverTypeStep ResolverType = "step"

	// ResolverTypeStage is a stage level resolver.
	ResolverTypeStage ResolverType = "stage"
)

func ParseResolverType(s string) (ResolverType, error) {
	switch s {
	case "step":
		return ResolverTypeStep, nil
	case "stage":
		return ResolverTypeStage, nil
	default:
		return "", fmt.Errorf("unknown template type provided: %s", s)
	}
}

func (t ResolverType) String() string {
	switch t {
	case ResolverTypeStep:
		return "step"
	case ResolverTypeStage:
		return "stage"
	default:
		return undefined
	}
}
