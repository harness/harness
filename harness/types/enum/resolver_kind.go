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

// ResolverKind represents the kind of resolver.
type ResolverKind string

const (
	// ResolverKindPlugin is a plugin resolver.
	ResolverKindPlugin ResolverKind = "plugin"

	// ResolverKindTemplate is a template resolver.
	ResolverKindTemplate ResolverKind = "template"
)

func ParseResolverKind(r string) (ResolverKind, error) {
	switch r {
	case "plugin":
		return ResolverKindPlugin, nil
	case "template":
		return ResolverKindTemplate, nil
	default:
		return "", fmt.Errorf("unknown resolver kind provided: %s", r)
	}
}

func (r ResolverKind) String() string {
	switch r {
	case ResolverKindPlugin:
		return "plugin"
	case ResolverKindTemplate:
		return "template"
	default:
		return undefined
	}
}
