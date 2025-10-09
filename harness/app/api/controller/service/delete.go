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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

/*
 * Delete deletes a service.
 */
func (c *Controller) Delete(ctx context.Context, session *auth.Session,
	serviceUID string) error {
	svc, err := findServiceFromUID(ctx, c.principalStore, serviceUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent
	if err = apiauth.CheckService(ctx, c.authorizer, session, svc, enum.PermissionServiceDelete); err != nil {
		return err
	}

	return c.principalStore.DeleteService(ctx, svc.ID)
}
