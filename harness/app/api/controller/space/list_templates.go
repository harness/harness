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

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListTemplates lists the templates in a space.
func (c *Controller) ListTemplates(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.ListQueryFilter,
) ([]*types.Template, int64, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find parent space: %w", err)
	}

	err = apiauth.CheckTemplate(
		ctx,
		c.authorizer,
		session,
		space.Path,
		"",
		enum.PermissionTemplateView,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("could not authorize: %w", err)
	}

	count, err := c.templateStore.Count(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count templates in the space: %w", err)
	}

	templates, err := c.templateStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, count, nil
}
