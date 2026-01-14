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

package interfaces

import (
	"context"

	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RegistryMetadataHelper provides helper methods for registry metadata operations.
type RegistryMetadataHelper interface {
	// GetRegistryRequestBaseInfo retrieves base info for registry request.
	GetRegistryRequestBaseInfo(
		ctx context.Context,
		parentPath string,
		identifier string,
	) (*registrytypes.RegistryRequestBaseInfo, error)

	// GetPermissionChecks returns permission checks for registry operations.
	GetPermissionChecks(
		space *types.SpaceCore,
		registryIdentifier string,
		permission enum.Permission,
	) []types.PermissionCheck

	// MapToWebhookCore maps webhook request to core webhook type.
	MapToWebhookCore(
		ctx context.Context,
		webhookRequest api.WebhookRequest,
		regInfo *registrytypes.RegistryRequestBaseInfo,
	) (*types.WebhookCore, error)

	// MapToWebhookResponseEntity maps webhook core to response entity.
	MapToWebhookResponseEntity(
		ctx context.Context,
		webhook *types.WebhookCore,
	) (*api.Webhook, error)

	// MapToInternalWebhookTriggers maps webhook triggers to internal type.
	MapToInternalWebhookTriggers(
		triggers []api.Trigger,
	) []enum.WebhookTrigger

	// MapToAPIWebhookTriggers maps webhook triggers to API type.
	MapToAPIWebhookTriggers(
		triggers []enum.WebhookTrigger,
	) []api.Trigger

	// GetSecretSpaceID retrieves secret space ID from secret space path.
	GetSecretSpaceID(
		ctx context.Context,
		secretSpacePath *string,
	) (int64, error)
}
