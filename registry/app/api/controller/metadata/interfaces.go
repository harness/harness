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

package metadata

import (
	"context"

	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type SpaceFinder interface {
	FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error)
	FindByID(ctx context.Context, spaceID int64) (*types.SpaceCore, error)
}

type RegistryMetadataHelper interface {
	GetPermissionChecks(
		space *types.SpaceCore,
		registryIdentifier string,
		permission enum.Permission,
	) []types.PermissionCheck
	GetRegistryRequestBaseInfo(
		ctx context.Context,
		parentRef string,
		regRef string,
	) (*RegistryRequestBaseInfo, error)
	getSecretSpaceID(ctx context.Context, secretSpacePath *string) (int, error)
	MapToAPIWebhookTriggers(triggers []enum.WebhookTrigger) []api.Trigger
	MapToInternalWebhookTriggers(
		triggers []api.Trigger,
	) []enum.WebhookTrigger
	MapToWebhookCore(
		ctx context.Context,
		webhookRequest api.WebhookRequest,
		regInfo *RegistryRequestBaseInfo,
	) (*types.WebhookCore, error)
	MapToWebhookResponseEntity(
		ctx context.Context,
		createdWebhook *types.WebhookCore,
	) (*api.Webhook, error)
}

type WebhookService interface {
	ReTriggerWebhookExecution(ctx context.Context, webhookExecutionID int64) (*gitnesswebhook.TriggerResult, error)
}
