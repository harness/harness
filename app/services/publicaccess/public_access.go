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

package publicaccess

import (
	"context"

	"github.com/harness/gitness/types"
)

// PublicAccess is an abstraction of an entity responsible for managing public access to resources.
type PublicAccess interface {
	/*
	 * Get returns whether public access is enabled on the resource.
	 * Returns
	 *		(true, nil)   - resource public access is allowed
	 *		(false, nil)  - resource public access is not allowed
	 *		(false, err)  - an error occurred while performing the public access check.
	 */
	Get(
		ctx context.Context,
		scope *types.Scope,
		resource *types.Resource,
	) (bool, error)

	/*
	 * Sets or deletes public access mode for the resource based on the value of 'enable'.
	 * Returns
	 *		err  - resource public access mode has been updated successfully
	 *		nil  - an error occurred while performing the public access set.
	 */
	Set(
		ctx context.Context,
		scope *types.Scope,
		resource *types.Resource,
		enable bool,
	) error
}