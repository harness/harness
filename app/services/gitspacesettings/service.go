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

type GitspaceSettingsService interface {
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

// Existing SettingsService struct implements GitspaceSettingsService
var _ GitspaceSettingsService = (*settingsService)(nil)

type settingsService struct {
	gitspaceSettingsStore store.GitspaceSettingsStore
}

func (s *settingsService) GetInfraProviderSettings(
	ctx context.Context,
	spaceID int64,
	criteria *types.GitspaceSettingsCriteria,
) (*types.InfraProviderSettings, error) {
	return nil, nil
}

func NewSettingsService(
	_ context.Context,
	store store.GitspaceSettingsStore,
) GitspaceSettingsService {
	return &settingsService{
		gitspaceSettingsStore: store,
	}
}

func (s *settingsService) GetGitspaceConfigSettings(
	ctx context.Context,
	spaceID int64,
	criteria *types.GitspaceSettingsCriteria,
) (*types.GitspaceConfigSettings, error) {
	return nil, nil
}

func (s *settingsService) ValidateGitspaceConfigCreate(
	ctx context.Context,
	resource types.InfraProviderResource,
	gitspaceConfig types.GitspaceConfig,
) error {
	return nil
}

func (s *settingsService) ValidateResolvedSCMDetails(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	scmResolvedDetails *scm.ResolvedDetails,
) *types.GitspaceError {
	return nil
}
