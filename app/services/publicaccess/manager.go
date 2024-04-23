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
	"errors"
	"fmt"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

type PublicAccessManager struct {
	publicResourceStore store.PublicResource
}

func NewPublicAccessManager(
	publicResourceStore store.PublicResource,
) *PublicAccessManager {
	return &PublicAccessManager{
		publicResourceStore: publicResourceStore,
	}
}

func (r *PublicAccessManager) Get(
	ctx context.Context,
	resource *types.PublicResource) (bool, error) {
	err := r.publicResourceStore.Find(ctx, resource)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get public access resource: %w", err)
	}

	return true, nil
}

func (r *PublicAccessManager) Set(ctx context.Context,
	resource *types.PublicResource,
	enable bool) error {
	if enable {
		err := r.publicResourceStore.Create(ctx, resource)
		if errors.Is(err, gitness_store.ErrDuplicate) {
			return nil
		}
		return err
	} else {
		return r.publicResourceStore.Delete(ctx, resource)
	}
}
