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
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/exporter"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
)

type ExportProgressOutput struct {
	Repos []job.Progress `json:"repos"`
}

// ExportProgress returns progress of the export job.
func (c *Controller) ExportProgress(ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (ExportProgressOutput, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return ExportProgressOutput{}, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView); err != nil {
		return ExportProgressOutput{}, err
	}

	progress, err := c.exporter.GetProgressForSpace(ctx, space.ID)
	if errors.Is(err, exporter.ErrNotFound) {
		return ExportProgressOutput{}, usererror.NotFound("No recent or ongoing export found for space.")
	}
	if err != nil {
		return ExportProgressOutput{}, fmt.Errorf("failed to retrieve export progress: %w", err)
	}

	return ExportProgressOutput{Repos: progress}, nil
}
