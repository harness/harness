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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"
)

type LinkedSourceOutput struct {
	URL string `json:"url"`
}

func (c *Controller) GetLinkedSource(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) (*LinkedSourceOutput, error) {
	repo, err := c.getRepoCheckAccessWithLinked(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	if repo.Type != enum.RepoTypeLinked {
		return nil, errors.InvalidArgument("Repository is not a linked repository.")
	}

	linkedRepo, err := c.linkedRepoStore.Find(ctx, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find linked repository (id=%d): %w", repo.ID, err)
	}

	connector := importer.ConnectorDef{
		Path:       linkedRepo.ConnectorPath,
		Identifier: linkedRepo.ConnectorIdentifier,
	}

	accessInfo, err := c.connectorService.GetAccessInfo(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("failed to get access info: %w", err)
	}

	if accessInfo.URL == "" {
		return nil, fmt.Errorf("connector returned an empty repository URL")
	}

	return &LinkedSourceOutput{URL: accessInfo.URL}, nil
}
