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

package gitspacesettings

import (
	"context"

	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

type Service interface {
	GetGitspaceConfigSettings(
		ctx context.Context,
		spaceID int64,
		criteria *types.GitspaceSettingsCriteria,
	) (*types.GitspaceConfigSettings, error)

	GetInfraProviderSettings(
		ctx context.Context,
		spaceID int64,
		criteria *types.GitspaceSettingsCriteria,
	) (*types.InfraProviderSettings, error)

	ValidateGitspaceConfigCreate(
		ctx context.Context,
		resource types.InfraProviderResource,
		gitspaceConfig types.GitspaceConfig,
	) error

	ValidateResolvedSCMDetails(
		ctx context.Context,
		gitspaceConfig types.GitspaceConfig,
		scmResolvedDetails *scm.ResolvedDetails,
	) *types.GitspaceError
}

// Existing SettingsService struct implements gitspacesettings.Service.
var _ Service = (*settingsService)(nil)

type settingsService struct {
	gitspaceSettingsStore store.GitspaceSettingsStore
}

func (s *settingsService) GetInfraProviderSettings(
	_ context.Context,
	_ int64,
	_ *types.GitspaceSettingsCriteria,
) (*types.InfraProviderSettings, error) {
	return nil, nil // nolint: nilnil
}

func NewSettingsService(
	_ context.Context,
	store store.GitspaceSettingsStore,
) Service {
	return &settingsService{
		gitspaceSettingsStore: store,
	}
}

func (s *settingsService) GetGitspaceConfigSettings(
	_ context.Context,
	_ int64,
	_ *types.GitspaceSettingsCriteria,
) (*types.GitspaceConfigSettings, error) {
	return nil, nil // nolint: nilnil
}

func (s *settingsService) ValidateGitspaceConfigCreate(
	_ context.Context,
	_ types.InfraProviderResource,
	_ types.GitspaceConfig,
) error {
	return nil
}

func (s *settingsService) ValidateResolvedSCMDetails(
	_ context.Context,
	_ types.GitspaceConfig,
	_ *scm.ResolvedDetails,
) *types.GitspaceError {
	return nil
}
