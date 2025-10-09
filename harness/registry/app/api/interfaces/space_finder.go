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

package interfaces

import (
	"context"

	"github.com/harness/gitness/types"
)

// SpaceFinder defines the interface for finding spaces.
type SpaceFinder interface {
	// FindByID finds the space by id.
	FindByID(ctx context.Context, id int64) (*types.SpaceCore, error)

	// FindByRef finds the space using the spaceRef as either the id or the space path.
	FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error)
}
