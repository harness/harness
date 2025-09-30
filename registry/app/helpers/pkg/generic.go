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

package pkg

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type GenericPackageType interface {
	interfaces.PackageHelper
}

type genericPackageType struct {
	packageType          string
	registryHelper       interfaces.RegistryHelper
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
}

func NewGenericPackageType(registryHelper interfaces.RegistryHelper) GenericPackageType {
	return &genericPackageType{
		packageType:    string(artifact.PackageTypeGENERIC),
		registryHelper: registryHelper,
		validRepoTypes: []string{
			string(artifact.RegistryTypeUPSTREAM),
			string(artifact.RegistryTypeVIRTUAL),
		},
		validUpstreamSources: []string{
			string(artifact.UpstreamConfigSourceCustom),
		},
		upstreamSourceConfig: map[string]UpstreamSourceConfig{
			string(artifact.UpstreamConfigSourceCustom): {
				urlRequired: true,
			},
		},
	}
}

func (c *genericPackageType) GetPackageType() string {
	return c.packageType
}

func (c *genericPackageType) IsValidRepoType(repoType string) bool {
	for _, validRepoType := range c.validRepoTypes {
		if validRepoType == repoType {
			return true
		}
	}
	return false
}

func (c *genericPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	for _, validUpstreamSource := range c.validUpstreamSources {
		if validUpstreamSource == upstreamSource {
			return true
		}
	}
	return false
}

func (c *genericPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *genericPackageType) GetPullCommand(_ string, _ string, _ string, _ string,
	_ string, _ string, _ bool) string {
	return ""
}

func (c *genericPackageType) DeleteImage() error {
	return fmt.Errorf("not implemented")
}

func (c *genericPackageType) DeleteVersion(ctx context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ *types.Image,
	_ string,
	_ string,
) error {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return fmt.Errorf("not implemented")
}

func (c *genericPackageType) ReportDeleteVersionEvent(ctx context.Context,
	_ int64,
	_ int64,
	_ string,
	_ string,
) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *genericPackageType) ReportBuildPackageIndexEvent(ctx context.Context, _ int64, _ string) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *genericPackageType) ReportBuildRegistryIndexEvent(ctx context.Context, _ int64, _ []types.SourceRef) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}
