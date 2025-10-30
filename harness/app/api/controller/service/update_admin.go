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

package service

import (
	"context"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateAdmin updates the admin state of a service.
func (c *Controller) UpdateAdmin(ctx context.Context, session *auth.Session,
	serviceUID string, admin bool) (*types.Service, error) {
	sbc, err := findServiceFromUID(ctx, c.principalStore, serviceUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckService(ctx, c.authorizer, session, sbc, enum.PermissionServiceEditAdmin); err != nil {
		return nil, err
	}

	sbc.Admin = admin
	sbc.Updated = time.Now().UnixMilli()

	err = c.principalStore.UpdateService(ctx, sbc)
	if err != nil {
		return nil, err
	}

	return sbc, nil
}
