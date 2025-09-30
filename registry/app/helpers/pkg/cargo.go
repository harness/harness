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
	"strings"

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/registry/types"
)

type CargoPackageType interface {
	interfaces.PackageHelper
}

type UpstreamSourceConfig struct {
	urlRequired bool
}

type cargoPackageType struct {
	packageType          string
	registryHelper       interfaces.RegistryHelper
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
}

func NewCargoPackageType(registryHelper interfaces.RegistryHelper) CargoPackageType {
	return &cargoPackageType{
		packageType:    string(artifact.PackageTypeCARGO),
		registryHelper: registryHelper,
		validRepoTypes: []string{
			string(artifact.RegistryTypeUPSTREAM),
			string(artifact.RegistryTypeVIRTUAL),
		},
		validUpstreamSources: []string{
			string(artifact.UpstreamConfigSourceCustom),
			string(artifact.UpstreamConfigSourceCrates),
		},
		upstreamSourceConfig: map[string]UpstreamSourceConfig{
			string(artifact.UpstreamConfigSourceCustom): {
				urlRequired: true,
			},
			string(artifact.UpstreamConfigSourceCrates): {
				urlRequired: false,
			},
		},
	}
}

func (c *cargoPackageType) GetPackageType() string {
	return c.packageType
}

func (c *cargoPackageType) IsValidRepoType(repoType string) bool {
	for _, validRepoType := range c.validRepoTypes {
		if validRepoType == repoType {
			return true
		}
	}
	return false
}

func (c *cargoPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	for _, validUpstreamSource := range c.validUpstreamSources {
		if validUpstreamSource == upstreamSource {
			return true
		}
	}
	return false
}

func (c *cargoPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *cargoPackageType) GetPullCommand(_ string, _ string, image string, version string,
	_ string, _ string, _ bool) string {
	downloadCommand := "cargo add <ARTIFACT>@<VERSION> --registry <REGISTRY>"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<ARTIFACT>": image,
		"<VERSION>":  version,
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func (c *cargoPackageType) DeleteImage() error {
	return nil
}

func (c *cargoPackageType) DeleteVersion(ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	imageInfo *types.Image,
	artifactName string,
	versionName string,
) error {
	err := c.registryHelper.DeleteVersion(ctx, regInfo, imageInfo, artifactName, versionName)
	if err != nil {
		return fmt.Errorf("failed to delete cargo artifact: %w", err)
	}
	return nil
}

func (c *cargoPackageType) ReportDeleteVersionEvent(ctx context.Context,
	principalID int64,
	registryID int64,
	artifactName string,
	version string,
) {
	payload := webhook.GetArtifactDeletedPayloadForCommonArtifacts(
		principalID,
		registryID,
		artifact.PackageTypeCARGO,
		artifactName,
		version,
	)
	c.registryHelper.ReportDeleteVersionEvent(ctx, &payload)
}

func (c *cargoPackageType) ReportBuildPackageIndexEvent(ctx context.Context, registryID int64, artifactName string) {
	c.registryHelper.ReportBuildPackageIndexEvent(ctx, registryID, artifactName)
}

func (c *cargoPackageType) ReportBuildRegistryIndexEvent(_ context.Context, _ int64, _ []types.SourceRef) {
	// no-op for cargo
}
