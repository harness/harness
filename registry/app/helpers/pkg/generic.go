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

type GenericPackageType interface {
	interfaces.PackageHelper
}

type genericPackageType struct {
	packageType          string
	registryHelper       interfaces.RegistryHelper
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
	pathPackageType      string
}

func NewGenericPackageType(registryHelper interfaces.RegistryHelper) GenericPackageType {
	return &genericPackageType{
		packageType:     string(artifact.PackageTypeGENERIC),
		pathPackageType: string(types.PathPackageTypeGeneric),
		registryHelper:  registryHelper,
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

func (c *genericPackageType) IsFileOperationSupported() bool {
	return true
}

func (c *genericPackageType) GetPathPackageType() string {
	return c.pathPackageType
}

func (c *genericPackageType) IsValidRepoType(repoType string) bool {
	return slices.Contains(c.validRepoTypes, repoType)
}

func (c *genericPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	return slices.Contains(c.validUpstreamSources, upstreamSource)
}

func (c *genericPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *genericPackageType) GetPullCommand(_ string, _ string, _ string) string {
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

func (c *genericPackageType) GetFilePath(
	_ string,
	_ string,
) string {
	return ""
}

func (c *genericPackageType) DeleteArtifact(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ string,
) error {
	return nil
}

func (c *genericPackageType) GetPackageURL(_ context.Context,
	_ string,
	_ string,
) string {
	return ""
}

func (c *genericPackageType) GetArtifactMetadata(
	_ types.ArtifactMetadata,
) *artifact.ArtifactMetadata {
	return nil
}

func (c *genericPackageType) GetArtifactVersionMetadata(
	_ string,
	_ types.NonOCIArtifactMetadata,
) *artifact.ArtifactVersionMetadata {
	return nil
}

func (c *genericPackageType) GetDownloadFileCommand(
	_ string,
	_ string,
	_ string,
	_ bool,
) string {
	return ""
}

func (c *genericPackageType) GetFileMetadata(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ types.FileNodeMetadata,
) *artifact.FileDetail {
	return nil
}

func (c *genericPackageType) GetArtifactDetail(
	_ *types.Image,
	_ *types.Artifact,
	_ int64,
) (*artifact.ArtifactDetail, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *genericPackageType) GetClientSetupDetails(
	_ context.Context,
	_ string,
	_ *artifact.ArtifactParam,
	_ *artifact.VersionParam,
	_ artifact.RegistryType,
) (*artifact.ClientSetupDetails, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *genericPackageType) BuildRegistryIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildRegistryIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *genericPackageType) BuildPackageIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *genericPackageType) BuildPackageMetadataAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageMetadataTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *genericPackageType) GetNodePathsForImage(
	_ *string,
	packageName string,
) ([]string, error) {
	return []string{"/" + packageName}, nil
}

func (c *genericPackageType) GetNodePathsForArtifact(
	_ *string,
	packageName string,
	version string,
) ([]string, error) {
	paths, err := c.GetNodePathsForImage(nil, packageName)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = path + "/" + version
	}
	return result, nil
}

func (c *genericPackageType) GetPkgDownloadURL(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
) (string, error) {
	return "", nil
}

func (c *genericPackageType) GetPurlForArtifact(
	_ string,
	_ string,
) (string, error) {
	return "", fmt.Errorf("PURL not supported for generic package type")
}
