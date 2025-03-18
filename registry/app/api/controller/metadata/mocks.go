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

//nolint:gocritic
import (
	"context"

	"github.com/harness/gitness/app/auth"
	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
)

type MockWebhooksRepository struct{ mock.Mock }
type MockRegistryMetadataHelper struct{ mock.Mock }
type MockWebhookService struct{ mock.Mock }
type MockAuthorizer struct{ mock.Mock }
type MockWebhooksExecutionRepository struct{ mock.Mock }
type MockSpaceFinder struct{ mock.Mock }
type MockRegistryRepository struct{ mock.Mock }
type MockSpacePathStore struct{ mock.Mock }

//nolint:errcheck
func (m *MockWebhookService) ReTriggerWebhookExecution(
	ctx context.Context,
	webhookExecutionID int64,
) (*gitnesswebhook.TriggerResult, error) {
	args := m.Called(ctx, webhookExecutionID)
	if args.Get(0) != nil {
		return args.Get(0).(*gitnesswebhook.TriggerResult), args.Error(1)
	}
	return nil, args.Error(1)
}

//nolint:errcheck
func (m *MockRegistryMetadataHelper) GetPermissionChecks(
	space *gitnesstypes.SpaceCore,
	registryIdentifier string,
	permission enum.Permission,
) []gitnesstypes.PermissionCheck {
	args := m.Called(space, registryIdentifier, permission)
	if args.Get(0) != nil {
		return args.Get(0).([]gitnesstypes.PermissionCheck)
	}
	return nil
}

//nolint:errcheck
func (m *MockRegistryMetadataHelper) GetRegistryRequestBaseInfo(
	ctx context.Context,
	parentRef string,
	regRef string,
) (*RegistryRequestBaseInfo, error) {
	args := m.Called(ctx, parentRef, regRef)
	if args.Get(0) != nil {
		return args.Get(0).(*RegistryRequestBaseInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRegistryMetadataHelper) getSecretSpaceID(_ context.Context, _ *string) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryMetadataHelper) MapToAPIWebhookTriggers(
	_ []enum.WebhookTrigger,
) []artifact.Trigger {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryMetadataHelper) MapToInternalWebhookTriggers(
	_ []artifact.Trigger,
) []enum.WebhookTrigger {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryMetadataHelper) MapToWebhookCore(
	_ context.Context,
	_ artifact.WebhookRequest,
	_ *RegistryRequestBaseInfo,
) (*gitnesstypes.WebhookCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryMetadataHelper) MapToWebhookResponseEntity(
	_ context.Context,
	_ *gitnesstypes.WebhookCore,
) (*artifact.Webhook, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockAuthorizer) Check(
	_ context.Context,
	_ *auth.Session,
	_ *gitnesstypes.Scope,
	_ *gitnesstypes.Resource,
	_ enum.Permission,
) (bool, error) {
	// TODO implement me
	panic("implement me")
}

//nolint:errcheck
func (m *MockAuthorizer) CheckAll(
	ctx context.Context,
	session *auth.Session,
	permissionChecks ...gitnesstypes.PermissionCheck,
) (bool, error) {
	args := m.Called(ctx, session, permissionChecks)
	return args.Get(0).(bool), args.Error(1)
}

//nolint:errcheck
func (m *MockWebhooksExecutionRepository) Find(
	ctx context.Context,
	id int64,
) (*gitnesstypes.WebhookExecutionCore, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*gitnesstypes.WebhookExecutionCore), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWebhooksExecutionRepository) Create(_ context.Context, _ *gitnesstypes.WebhookExecutionCore) error {
	// TODO implement me
	panic("implement me")
}

//nolint:errcheck
func (m *MockWebhooksExecutionRepository) ListForWebhook(
	ctx context.Context,
	webhookID int64,
	limit int,
	page int,
	size int,
) ([]*gitnesstypes.WebhookExecutionCore, error) {
	args := m.Called(ctx, webhookID, limit, page, size)
	if args.Get(0) != nil {
		return args.Get(0).([]*gitnesstypes.WebhookExecutionCore), args.Error(1)
	}
	return nil, args.Error(1)
}

//nolint:errcheck
func (m *MockWebhooksExecutionRepository) CountForWebhook(ctx context.Context, webhookID int64) (int64, error) {
	args := m.Called(ctx, webhookID)
	if args.Get(1) == nil {
		return args.Get(0).(int64), nil
	}
	return 0, args.Error(1)
}

func (m *MockWebhooksExecutionRepository) ListForTrigger(
	_ context.Context,
	_ string,
) ([]*gitnesstypes.WebhookExecutionCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) Create(_ context.Context, _ *gitnesstypes.WebhookCore) error {
	// TODO implement me
	panic("implement me")
}

//nolint:errcheck
func (m *MockWebhooksRepository) GetByRegistryAndIdentifier(
	ctx context.Context,
	registryID int64,
	webhookIdentifier string,
) (*gitnesstypes.WebhookCore, error) {
	args := m.Called(ctx, registryID, webhookIdentifier)
	if args.Get(0) != nil {
		return args.Get(0).(*gitnesstypes.WebhookCore), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWebhooksRepository) Find(_ context.Context, _ int64) (*gitnesstypes.WebhookCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) ListByRegistry(
	_ context.Context,
	_ string,
	_ string,
	_ int,
	_ int,
	_ string,
	_ int64,
) ([]*gitnesstypes.WebhookCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) ListAllByRegistry(
	_ context.Context,
	_ []gitnesstypes.WebhookParentInfo,
) ([]*gitnesstypes.WebhookCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) CountAllByRegistry(
	_ context.Context, _ int64, _ string,
) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) Update(_ context.Context, _ *gitnesstypes.WebhookCore) error {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) DeleteByRegistryAndIdentifier(
	_ context.Context, _ int64, _ string,
) error {
	// TODO implement me
	panic("implement me")
}

func (m *MockWebhooksRepository) UpdateOptLock(
	_ context.Context, _ *gitnesstypes.WebhookCore, _ func(hook *gitnesstypes.WebhookCore) error,
) (*gitnesstypes.WebhookCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockSpaceFinder) FindByID(_ context.Context, _ int64) (*gitnesstypes.SpaceCore, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) Get(_ context.Context, _ int64) (repository *types.Registry, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) GetByIDIn(_ context.Context, _ []int64) (registries *[]types.Registry, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) GetByRootParentIDAndName(
	_ context.Context, _ int64, _ string,
) (registry *types.Registry, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) Create(_ context.Context, _ *types.Registry) (id int64, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) Delete(_ context.Context, _ int64, _ string) (err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) Update(_ context.Context, _ *types.Registry) (err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) GetAll(
	_ context.Context, _ int64, _ []string, _ string, _ string, _ int, _ int, _ string, _ string, _ bool,
) (repos *[]store.RegistryMetadata, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) CountAll(
	_ context.Context, _ int64, _ []string, _ string, _ string,
) (count int64, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) FetchUpstreamProxyIDs(
	_ context.Context,
	_ []string,
	_ int64,
) (ids []int64, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) FetchRegistriesIDByUpstreamProxyID(
	_ context.Context, _ string, _ int64,
) (ids []int64, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) FetchUpstreamProxyKeys(_ context.Context, _ []int64) (repokeys []string, err error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockRegistryRepository) Count(_ context.Context) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockSpacePathStore) InsertSegment(_ context.Context, _ *gitnesstypes.SpacePathSegment) error {
	// TODO implement me
	panic("implement me")
}

//nolint:errcheck
func (m *MockSpacePathStore) FindByPath(ctx context.Context, path string) (*gitnesstypes.SpacePath, error) {
	args := m.Called(ctx, path)
	if args.Get(0) != nil {
		return args.Get(0).(*gitnesstypes.SpacePath), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSpacePathStore) DeletePrimarySegment(_ context.Context, _ int64) error {
	// TODO implement me
	panic("implement me")
}

func (m *MockSpacePathStore) DeletePathsAndDescendandPaths(_ context.Context, _ int64) error {
	// TODO implement me
	panic("implement me")
}

//nolint:errcheck
func (m *MockSpaceFinder) FindByRef(ctx context.Context, ref string) (*gitnesstypes.SpaceCore, error) {
	args := m.Called(ctx, ref)
	if args.Get(0) != nil {
		return args.Get(0).(*gitnesstypes.SpaceCore), args.Error(1)
	}
	return nil, args.Error(1)
}

//nolint:errcheck
func (m *MockRegistryRepository) GetByParentIDAndName(
	ctx context.Context,
	parentID int64,
	name string,
) (*types.Registry, error) {
	args := m.Called(ctx, parentID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*types.Registry), args.Error(1)
	}
	return nil, args.Error(1)
}

//nolint:errcheck
func (m *MockSpacePathStore) FindPrimaryBySpaceID(ctx context.Context, spaceID int64) (*gitnesstypes.SpacePath, error) {
	args := m.Called(ctx, spaceID)
	if args.Get(0) != nil {
		return args.Get(0).(*gitnesstypes.SpacePath), args.Error(1)
	}
	return nil, args.Error(1)
}
