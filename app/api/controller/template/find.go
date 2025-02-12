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

package template

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	resolverType enum.ResolverType,
) (*types.Template, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckTemplate(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionTemplateView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	template, err := c.templateStore.FindByIdentifierAndType(ctx, space.ID, identifier, resolverType)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	return template, nil
}
