// Copyright 2026 Harness, Inc.
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

package infraprovider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz/authztest"
	serviceinfraprovider "github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type staticSpaceIDCache struct {
	spaces map[int64]*types.SpaceCore
}

func (c *staticSpaceIDCache) Stats() (int64, int64)            { return 0, 0 }
func (c *staticSpaceIDCache) Evict(_ context.Context, _ int64) {}
func (c *staticSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if space, ok := c.spaces[id]; ok {
		return space, nil
	}
	return nil, fmt.Errorf("space %d not found", id)
}

type recordingInfraProviderConfigStore struct {
	findByIdentifierCalls int
}

func (s *recordingInfraProviderConfigStore) Find(
	context.Context, int64, bool,
) (*types.InfraProviderConfig, error) {
	return nil, errors.New("unexpected config store call")
}

func (s *recordingInfraProviderConfigStore) FindByType(
	context.Context, int64, enum.InfraProviderType, bool,
) (*types.InfraProviderConfig, error) {
	return nil, errors.New("unexpected config store call")
}

func (s *recordingInfraProviderConfigStore) FindByIdentifier(
	context.Context, int64, string,
) (*types.InfraProviderConfig, error) {
	s.findByIdentifierCalls++
	return nil, errors.New("unexpected config store call")
}

func (s *recordingInfraProviderConfigStore) List(
	context.Context, *types.InfraProviderConfigFilter,
) ([]*types.InfraProviderConfig, error) {
	return nil, errors.New("unexpected config store call")
}

func (s *recordingInfraProviderConfigStore) Create(context.Context, *types.InfraProviderConfig) error {
	return errors.New("unexpected config store call")
}

func (s *recordingInfraProviderConfigStore) Update(context.Context, *types.InfraProviderConfig) error {
	return errors.New("unexpected config store call")
}

func (s *recordingInfraProviderConfigStore) Delete(context.Context, int64) error {
	return errors.New("unexpected config store call")
}

func TestFindRequiresInfraProviderViewPermission(t *testing.T) {
	configStore := &recordingInfraProviderConfigStore{}
	spaceFinder := refcache.NewSpaceFinder(
		&staticSpaceIDCache{
			spaces: map[int64]*types.SpaceCore{
				1: {ID: 1, Path: "acme/platform"},
			},
		},
		nil,
		nil,
		storecache.Evictor[*types.SpaceCore]{},
	)
	infraProviderService := serviceinfraprovider.NewService(
		nil,
		nil,
		nil,
		configStore,
		nil,
		nil,
		spaceFinder,
		nil,
	)
	controller := NewController(authztest.DenyAuthorizer{}, spaceFinder, infraProviderService)

	session := &auth.Session{Principal: types.Principal{UID: "user"}}
	_, err := controller.Find(context.Background(), session, "1", "docker")
	if !errors.Is(err, apiauth.ErrForbidden) {
		t.Fatalf("Find() error = %v, want forbidden", err)
	}
	if configStore.findByIdentifierCalls != 0 {
		t.Fatalf("config store was called %d times after authorization failed", configStore.findByIdentifierCalls)
	}
}
