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
	"strings"

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type MavenPackageType interface {
	interfaces.PackageHelper
}

type mavenPackageType struct {
	packageType          string
	registryHelper       interfaces.RegistryHelper
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
	pathPackageType      string
}

func NewMavenPackageType(registryHelper interfaces.RegistryHelper) MavenPackageType {
	return &mavenPackageType{
		packageType:     string(artifact.PackageTypeMAVEN),
		pathPackageType: string(types.PathPackageTypeMaven),
		registryHelper:  registryHelper,
		validRepoTypes: []string{
			string(artifact.RegistryTypeUPSTREAM),
			string(artifact.RegistryTypeVIRTUAL),
		},
		validUpstreamSources: []string{
			string(artifact.UpstreamConfigSourceCustom),
			string(artifact.UpstreamConfigSourceMavenCentral),
		},
		upstreamSourceConfig: map[string]UpstreamSourceConfig{
			string(artifact.UpstreamConfigSourceCustom): {
				urlRequired: true,
			},
			string(artifact.UpstreamConfigSourceMavenCentral): {
				urlRequired: false,
			},
		},
	}
}

func (c *mavenPackageType) GetPackageType() string {
	return c.packageType
}

func (c *mavenPackageType) GetPathPackageType() string {
	return c.pathPackageType
}

func (c *mavenPackageType) IsValidRepoType(repoType string) bool {
	return slices.Contains(c.validRepoTypes, repoType)
}

func (c *mavenPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	return slices.Contains(c.validUpstreamSources, upstreamSource)
}

func (c *mavenPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *mavenPackageType) GetPullCommand(_ string, _ string, _ string) string {
	return ""
}

func (c *mavenPackageType) DeleteImage() error {
	return fmt.Errorf("not implemented")
}

func (c *mavenPackageType) DeleteVersion(ctx context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ *types.Image,
	_ string,
	_ string,
) error {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return fmt.Errorf("not implemented")
}

func (c *mavenPackageType) ReportDeleteVersionEvent(ctx context.Context,
	_ int64,
	_ int64,
	_ string,
	_ string,
) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *mavenPackageType) ReportBuildPackageIndexEvent(ctx context.Context, _ int64, _ string) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *mavenPackageType) ReportBuildRegistryIndexEvent(ctx context.Context, _ int64, _ []types.SourceRef) {
	log.Error().Ctx(ctx).Msg("Not implemented")
}

func (c *mavenPackageType) GetFilePath(
	_ string,
	_ string,
) string {
	return ""
}

func (c *mavenPackageType) DeleteArtifact(
	_ context.Context,
	_ *types.RegistryRequestBaseInfo,
	_ string,
) error {
	return nil
}

func (c *mavenPackageType) GetPackageURL(_ context.Context,
	_ string,
	_ string,
) string {
	return ""
}

func (c *mavenPackageType) GetArtifactMetadata(
	_ types.ArtifactMetadata,
) *artifact.ArtifactMetadata {
	return nil
}

func (c *mavenPackageType) GetArtifactVersionMetadata(
	_ string,
	_ types.NonOCIArtifactMetadata,
) *artifact.ArtifactVersionMetadata {
	return nil
}

func (c *mavenPackageType) GetDownloadFileCommand(
	_ string,
	_ string,
	_ string,
	_ bool,
) string {
	return ""
}

func (c *mavenPackageType) GetFileMetadata(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ types.FileNodeMetadata,
) *artifact.FileDetail {
	return nil
}

func (c *mavenPackageType) GetArtifactDetail(
	_ *types.Image,
	_ *types.Artifact,
	_ int64,
) (*artifact.ArtifactDetail, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *mavenPackageType) GetClientSetupDetails(
	_ context.Context,
	_ string,
	_ *artifact.ArtifactParam,
	_ *artifact.VersionParam,
	_ artifact.RegistryType,
) (*artifact.ClientSetupDetails, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *mavenPackageType) BuildRegistryIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildRegistryIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *mavenPackageType) BuildPackageIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *mavenPackageType) BuildPackageMetadataAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageMetadataTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *mavenPackageType) GetNodePathsForImage(
	_ *string,
	packageName string,
) ([]string, error) {
	parts := strings.SplitN(packageName, ":", 2)
	path := ""
	if len(parts) == 2 {
		groupID := strings.ReplaceAll(parts[0], ".", "/")
		path = groupID + "/" + parts[1]
	}
	return []string{"/" + path}, nil
}

func (c *mavenPackageType) GetNodePathsForArtifact(
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

func (c *mavenPackageType) GetPkgDownloadURL(
	ctx context.Context,
	rootIdentifier string,
	registryIdentifier string,
	_ string,
	_ string,
	_ string,
	_ string,
	filepath string,
) (string, error) {
	if filepath == "" {
		return "", fmt.Errorf("filepath cannot be empty")
	}

	baseURL := c.registryHelper.GetPackageURL(ctx, rootIdentifier, registryIdentifier, c.pathPackageType)

	downloadURL := fmt.Sprintf("%s%s", baseURL, filepath)
	return downloadURL, nil
}
