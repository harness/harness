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
	"slices"

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type NPMPackageType interface {
	interfaces.PackageHelper
}

type npmPackageType struct {
	packageType          string
	registryHelper       interfaces.RegistryHelper
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
	pathPackageType      string
}

func NewNPMPackageType(registryHelper interfaces.RegistryHelper) NPMPackageType {
	return &npmPackageType{
		packageType:     string(artifact.PackageTypeNPM),
		registryHelper:  registryHelper,
		pathPackageType: string(types.PathPackageTypeNpm),
		validRepoTypes: []string{
			string(artifact.RegistryTypeUPSTREAM),
			string(artifact.RegistryTypeVIRTUAL),
		},
		validUpstreamSources: []string{
			string(artifact.UpstreamConfigSourceCustom),
			string(artifact.UpstreamConfigSourceNpmJs),
		},
		upstreamSourceConfig: map[string]UpstreamSourceConfig{
			string(artifact.UpstreamConfigSourceCustom): {
				urlRequired: true,
			},
			string(artifact.UpstreamConfigSourceNpmJs): {
				urlRequired: false,
			},
		},
	}
}

func (c *npmPackageType) GetPackageType() string {
	return c.packageType
}

func (c *npmPackageType) GetPathPackageType() string {
	return c.pathPackageType
}

func (c *npmPackageType) IsValidRepoType(repoType string) bool {
	return slices.Contains(c.validRepoTypes, repoType)
}

func (c *npmPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	return slices.Contains(c.validUpstreamSources, upstreamSource)
}

func (c *npmPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *npmPackageType) GetPullCommand(_ string, _ string, _ string) string {
	return ""
}

func (c *npmPackageType) DeleteImage() error {
	return fmt.Errorf("not implemented")
}

func (c *npmPackageType) DeleteVersion(ctx context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ *types.Image,
	_ string,
	_ string,
) error {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return fmt.Errorf("not implemented")
}

func (c *npmPackageType) ReportDeleteVersionEvent(ctx context.Context,
	_ int64,
	_ int64,
	_ string,
	_ string,
) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *npmPackageType) ReportBuildPackageIndexEvent(ctx context.Context, _ int64, _ string) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *npmPackageType) ReportBuildRegistryIndexEvent(ctx context.Context, _ int64, _ []types.SourceRef) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *npmPackageType) GetFilePath(
	_ string,
	_ string,
) string {
	return ""
}

func (c *npmPackageType) DeleteArtifact(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ string,
) error {
	return nil
}

func (c *npmPackageType) GetPackageURL(_ context.Context,
	_ string,
	_ string,
) string {
	return ""
}

func (c *npmPackageType) GetArtifactMetadata(
	_ types.ArtifactMetadata,
) *artifact.ArtifactMetadata {
	return nil
}

func (c *npmPackageType) GetArtifactVersionMetadata(
	_ string,
	_ types.NonOCIArtifactMetadata,
) *artifact.ArtifactVersionMetadata {
	return nil
}

func (c *npmPackageType) GetDownloadFileCommand(
	_ string,
	_ string,
	_ string,
	_ bool,
) string {
	return ""
}

func (c *npmPackageType) GetFileMetadata(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ types.FileNodeMetadata,
) *artifact.FileDetail {
	return nil
}

func (c *npmPackageType) GetArtifactDetail(
	_ *types.Image,
	_ *types.Artifact,
	_ int64,
) (*artifact.ArtifactDetail, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *npmPackageType) GetClientSetupDetails(
	_ context.Context,
	_ string,
	_ *artifact.ArtifactParam,
	_ *artifact.VersionParam,
	_ artifact.RegistryType,
) (*artifact.ClientSetupDetails, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *npmPackageType) BuildRegistryIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildRegistryIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *npmPackageType) BuildPackageIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *npmPackageType) BuildPackageMetadataAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageMetadataTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *npmPackageType) GetNodePathsForImage(
	_ *string,
	packageName string,
) []string {
	return []string{"/" + packageName}
}

func (c *npmPackageType) GetNodePathsForArtifact(
	_ *string,
	packageName string,
	version string,
) []string {
	paths := c.GetNodePathsForImage(nil, packageName)
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = path + "/" + version
	}
	return result
}
