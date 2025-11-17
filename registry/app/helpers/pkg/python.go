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

type PythonPackageType interface {
	interfaces.PackageHelper
}

type pythonPackageType struct {
	packageType          string
	registryHelper       interfaces.RegistryHelper
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
	pathPackageType      string
}

func NewPythonPackageType(registryHelper interfaces.RegistryHelper) PythonPackageType {
	return &pythonPackageType{
		packageType:     string(artifact.PackageTypePYTHON),
		registryHelper:  registryHelper,
		pathPackageType: string(types.PathPackageTypePython),
		validRepoTypes: []string{
			string(artifact.RegistryTypeUPSTREAM),
			string(artifact.RegistryTypeVIRTUAL),
		},
		validUpstreamSources: []string{
			string(artifact.UpstreamConfigSourceCustom),
			string(artifact.UpstreamConfigSourcePyPi),
		},
		upstreamSourceConfig: map[string]UpstreamSourceConfig{
			string(artifact.UpstreamConfigSourceCustom): {
				urlRequired: true,
			},
			string(artifact.UpstreamConfigSourcePyPi): {
				urlRequired: false,
			},
		},
	}
}

func (c *pythonPackageType) GetPackageType() string {
	return c.packageType
}

func (c *pythonPackageType) GetPathPackageType() string {
	return c.pathPackageType
}

func (c *pythonPackageType) IsValidRepoType(repoType string) bool {
	return slices.Contains(c.validRepoTypes, repoType)
}

func (c *pythonPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	return slices.Contains(c.validUpstreamSources, upstreamSource)
}

func (c *pythonPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *pythonPackageType) GetPullCommand(_ string, _ string, _ string) string {
	return ""
}

func (c *pythonPackageType) DeleteImage() error {
	return fmt.Errorf("not implemented")
}

func (c *pythonPackageType) DeleteVersion(ctx context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ *types.Image,
	_ string,
	_ string,
) error {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return fmt.Errorf("not implemented")
}

func (c *pythonPackageType) ReportDeleteVersionEvent(ctx context.Context,
	_ int64,
	_ int64,
	_ string,
	_ string,
) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *pythonPackageType) ReportBuildPackageIndexEvent(ctx context.Context, _ int64, _ string) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *pythonPackageType) ReportBuildRegistryIndexEvent(ctx context.Context, _ int64, _ []types.SourceRef) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *pythonPackageType) GetFilePath(
	_ string,
	_ string,
) string {
	return ""
}

func (c *pythonPackageType) DeleteArtifact(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ string,
) error {
	return nil
}

func (c *pythonPackageType) GetPackageURL(_ context.Context,
	_ string,
	_ string,
) string {
	return ""
}

func (c *pythonPackageType) GetArtifactMetadata(
	_ types.ArtifactMetadata,
) *artifact.ArtifactMetadata {
	return nil
}

func (c *pythonPackageType) GetArtifactVersionMetadata(
	_ string,
	_ types.NonOCIArtifactMetadata,
) *artifact.ArtifactVersionMetadata {
	return nil
}

func (c *pythonPackageType) GetDownloadFileCommand(
	_ string,
	_ string,
	_ string,
	_ bool,
) string {
	return ""
}

func (c *pythonPackageType) GetFileMetadata(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ types.FileNodeMetadata,
) *artifact.FileDetail {
	return nil
}

func (c *pythonPackageType) GetArtifactDetail(
	_ *types.Image,
	_ *types.Artifact,
	_ int64,
) (*artifact.ArtifactDetail, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *pythonPackageType) GetClientSetupDetails(
	_ context.Context,
	_ string,
	_ *artifact.ArtifactParam,
	_ *artifact.VersionParam,
	_ artifact.RegistryType,
) (*artifact.ClientSetupDetails, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *pythonPackageType) BuildRegistryIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildRegistryIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *pythonPackageType) BuildPackageIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *pythonPackageType) BuildPackageMetadataAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageMetadataTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *pythonPackageType) GetNodePathsForImage(
	_ *string,
	packageName string,
) ([]string, error) {
	return []string{"/" + packageName}, nil
}

func (c *pythonPackageType) GetNodePathsForArtifact(
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
