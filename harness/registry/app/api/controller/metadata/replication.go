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

package metadata

import (
	"context"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

func (c *APIController) ListReplicationRules(
	_ context.Context,
	_ artifact.ListReplicationRulesRequestObject,
) (artifact.ListReplicationRulesResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) CreateReplicationRule(
	_ context.Context,
	_ artifact.CreateReplicationRuleRequestObject,
) (artifact.CreateReplicationRuleResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) DeleteReplicationRule(
	_ context.Context,
	_ artifact.DeleteReplicationRuleRequestObject,
) (artifact.DeleteReplicationRuleResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) GetReplicationRule(
	_ context.Context,
	_ artifact.GetReplicationRuleRequestObject,
) (artifact.GetReplicationRuleResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) UpdateReplicationRule(
	_ context.Context,
	_ artifact.UpdateReplicationRuleRequestObject,
) (artifact.UpdateReplicationRuleResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) ListMigrationImages(
	_ context.Context,
	_ artifact.ListMigrationImagesRequestObject,
) (artifact.ListMigrationImagesResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) GetMigrationLogsForImage(
	_ context.Context,
	_ artifact.GetMigrationLogsForImageRequestObject,
) (artifact.GetMigrationLogsForImageResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) StartMigration(
	_ context.Context,
	_ artifact.StartMigrationRequestObject,
) (artifact.StartMigrationResponseObject, error) {
	// TODO implement me
	panic("implement me")
}

func (c *APIController) StopMigration(
	_ context.Context,
	_ artifact.StopMigrationRequestObject,
) (artifact.StopMigrationResponseObject, error) {
	// TODO implement me
	panic("implement me")
}
