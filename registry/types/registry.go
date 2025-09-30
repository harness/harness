//  Copyright 2023 Harness, Inc.
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

package types

import (
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

// Registry DTO object.
type Registry struct {
	ID              int64
	UUID            string
	Name            string
	ParentID        int64
	RootParentID    int64
	Description     string
	Type            artifact.RegistryType
	PackageType     artifact.PackageType
	UpstreamProxies []int64
	AllowedPattern  []string
	BlockedPattern  []string
	Labels          []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       int64
	UpdatedBy       int64
	IsPublic        bool
}

func (r Registry) Identifier() int64 { return r.ID }
