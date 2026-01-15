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
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/utils/cargo"
	"github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/registry/types"
	registryutils "github.com/harness/gitness/registry/utils"
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
	pathPackageType      string
	validRepoTypes       []string
	validUpstreamSources []string
	upstreamSourceConfig map[string]UpstreamSourceConfig
	cargoRegistryHelper  cargo.RegistryHelper
}

func NewCargoPackageType(
	registryHelper interfaces.RegistryHelper,
	cargoRegistryHelper cargo.RegistryHelper,
) CargoPackageType {
	return &cargoPackageType{
		packageType:     string(artifact.PackageTypeCARGO),
		pathPackageType: string(types.PathPackageTypeCargo),
		registryHelper:  registryHelper,
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
		cargoRegistryHelper: cargoRegistryHelper,
	}
}

func (c *cargoPackageType) GetPackageType() string {
	return c.packageType
}

func (c *cargoPackageType) GetPathPackageType() string {
	return c.pathPackageType
}

func (c *cargoPackageType) IsValidRepoType(repoType string) bool {
	return slices.Contains(c.validRepoTypes, repoType)
}

func (c *cargoPackageType) IsValidUpstreamSource(upstreamSource string) bool {
	return slices.Contains(c.validUpstreamSources, upstreamSource)
}

func (c *cargoPackageType) IsURLRequiredForUpstreamSource(upstreamSource string) bool {
	config, ok := c.upstreamSourceConfig[upstreamSource]
	if !ok {
		return true
	}
	return config.urlRequired
}

func (c *cargoPackageType) GetPullCommand(_ string, image string, version string) string {
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

func (c *cargoPackageType) GetDownloadFileCommand(
	regURL string,
	artifact string,
	version string,
	isAnonymous bool,
) string {
	var authHeader string
	if !isAnonymous {
		authHeader = " --header '<AUTH_HEADER_PREFIX> <API_KEY>'"
	}
	downloadCommand := "curl --location '<HOSTNAME>/api/v1/crates/<ARTIFACT>/<VERSION>/download'" + authHeader +
		" -J -o '<OUTPUT_FILE_NAME>'"

	// Replace the placeholders with the actual values
	replacements := map[string]string{
		"<HOSTNAME>":           regURL,
		"<ARTIFACT>":           artifact,
		"<VERSION>":            version,
		"<AUTH_HEADER_PREFIX>": c.registryHelper.GetAuthHeaderPrefix(),
	}

	for placeholder, value := range replacements {
		downloadCommand = strings.ReplaceAll(downloadCommand, placeholder, value)
	}

	return downloadCommand
}

func (c *cargoPackageType) DeleteVersion(ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	imageInfo *types.Image,
	artifactName string,
	versionName string,
) error {
	err := c.registryHelper.DeleteVersion(
		ctx, regInfo, imageInfo, artifactName, versionName,
		c.GetFilePath(artifactName, versionName),
	)
	if err != nil {
		return fmt.Errorf("failed to delete cargo artifact version: %w", err)
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

func (c *cargoPackageType) GetFilePath(
	artifactName string,
	versionName string,
) string {
	filePathPrefix := "/crates/" + artifactName
	if versionName != "" {
		filePathPrefix += "/" + versionName
	}
	return filePathPrefix
}

func (c *cargoPackageType) DeleteArtifact(ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	artifactName string,
) error {
	filePath := c.GetFilePath(artifactName, "")
	err := c.registryHelper.DeleteGenericImage(ctx, regInfo, artifactName, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete cargo artifact: %w", err)
	}
	return nil
}

func (c *cargoPackageType) GetPackageURL(ctx context.Context,
	rootIdentifier string,
	registryIdentifier string,
) string {
	return c.registryHelper.GetPackageURL(ctx, rootIdentifier, registryIdentifier, "cargo")
}

func (c *cargoPackageType) GetArtifactMetadata(
	artifact types.ArtifactMetadata,
) *artifact.ArtifactMetadata {
	pullCommand := c.GetPullCommand("", artifact.Name, artifact.Version)
	return c.registryHelper.GetArtifactMetadata(artifact, pullCommand)
}

func (c *cargoPackageType) GetArtifactVersionMetadata(
	image string,
	tag types.NonOCIArtifactMetadata,
) *artifact.ArtifactVersionMetadata {
	pullCommand := c.GetPullCommand("", image, tag.Name)
	return c.registryHelper.GetArtifactVersionMetadata(tag, pullCommand, c.packageType)
}

func (c *cargoPackageType) GetFileMetadata(
	ctx context.Context,
	rootIdentifier string,
	registryIdentifier string,
	artifactName string,
	version string,
	file types.FileNodeMetadata,
) *artifact.FileDetail {
	filePathPrefix := "/crates/" + artifactName + "/" + version + "/"
	filename := strings.Replace(file.Path, filePathPrefix, "", 1)
	regURL := c.GetPackageURL(ctx, rootIdentifier, registryIdentifier)
	session, _ := request.AuthSessionFrom(ctx)
	downloadCommand := c.GetDownloadFileCommand(regURL, artifactName, version, auth.IsAnonymousSession(session))
	return c.registryHelper.GetFileMetadata(file, filename, downloadCommand)
}

func (c *cargoPackageType) GetArtifactDetail(
	img *types.Image,
	art *types.Artifact,
	downloadCount int64,
) (*artifact.ArtifactDetail, error) {
	var result map[string]any
	err := json.Unmarshal(art.Metadata, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	artifactDetails := c.registryHelper.GetArtifactDetail(img, art, result, downloadCount)
	if artifactDetails == nil {
		return nil, fmt.Errorf("failed to get artifact details")
	}
	err = artifactDetails.FromCargoArtifactDetailConfig(artifact.CargoArtifactDetailConfig{
		Metadata: &result,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact details: %w", err)
	}
	return artifactDetails, nil
}

func (c *cargoPackageType) GetClientSetupDetails(
	ctx context.Context,
	regRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) (*artifact.ClientSetupDetails, error) {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken

	registryURL := c.GetPackageURL(ctx, regRef, "")
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails

	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.GetAnonymousClientSetupDetails(
			registryType, staticStepType,
		)
	} else {
		clientSetupDetails = c.GetAuthenticatedClientSetupDetails(
			registryType, staticStepType, generateTokenType,
		)
	}
	c.registryHelper.ReplacePlaceholders(
		ctx, &clientSetupDetails.Sections, username, regRef, image, tag, registryURL, "", "", "")

	return &clientSetupDetails, nil
}

func (c *cargoPackageType) GetAuthenticatedClientSetupDetails(
	registryType artifact.RegistryType,
	staticStepType artifact.ClientSetupStepType,
	generateTokenType artifact.ClientSetupStepType,
) artifact.ClientSetupDetails {
	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: registryutils.StringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: registryutils.StringPtr("Create or update ~/.cargo/config.toml with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: registryutils.StringPtr("[registry]\n" +
							`global-credential-providers = ["cargo:token", "cargo:libsecret", "cargo:macos-keychain", "cargo:wincred"]` +
							"\n\n" +
							"[registries.harness-<REGISTRY_NAME>]\n" +
							`index = "sparse+<REGISTRY_URL>/index/"`),
					},
				},
			},
			{
				Header: registryutils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
			{
				Header: registryutils.StringPtr("Create or update ~/.cargo/credentials.toml with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: registryutils.StringPtr(
							"[registries.harness-<REGISTRY_NAME>]" + "\n" + `token = "Bearer <token from step 2>"`,
						),
					},
				},
			},
		},
	})

	// Publish section
	section2 := artifact.ClientSetupSection{
		Header: registryutils.StringPtr("Publish Package"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: registryutils.StringPtr("Publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: registryutils.StringPtr("cargo publish --registry harness-<REGISTRY_NAME>"),
					},
				},
			},
		},
	})

	// Install section
	section3 := getInstallPackageClientSetupSection(staticStepType)

	sections := []artifact.ClientSetupSection{
		section1,
		section2,
		section3,
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section1,
			section3,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Cargo Client Setup",
		SecHeader:  "Follow these instructions to install/use cargo packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *cargoPackageType) GetAnonymousClientSetupDetails(
	registryType artifact.RegistryType,
	staticStepType artifact.ClientSetupStepType,
) artifact.ClientSetupDetails {
	section1 := artifact.ClientSetupSection{
		Header: registryutils.StringPtr("Configure registry"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: registryutils.StringPtr("Create or update ~/.cargo/config.toml with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: registryutils.StringPtr("[registry]\n" +
							`global-credential-providers = ["cargo:token", "cargo:libsecret", "cargo:macos-keychain", "cargo:wincred"]` +
							"\n\n" +
							"[registries.harness-<REGISTRY_NAME>]\n" +
							`index = "sparse+<REGISTRY_URL>/index/"`),
					},
				},
			},
		},
	})

	// Install section
	section3 := getInstallPackageClientSetupSection(staticStepType)

	sections := []artifact.ClientSetupSection{
		section1,
		section3,
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section1,
			section3,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Cargo Client Setup",
		SecHeader:  "Follow these instructions to install/use cargo packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func getInstallPackageClientSetupSection(
	staticStepType artifact.ClientSetupStepType,
) artifact.ClientSetupSection {
	// Install section
	section3 := artifact.ClientSetupSection{
		Header: registryutils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: registryutils.StringPtr("Install a package using cargo"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: registryutils.StringPtr("cargo add <ARTIFACT_NAME>@<VERSION> --registry harness-<REGISTRY_NAME>"),
					},
				},
			},
		},
	})
	return section3
}

func (c *cargoPackageType) BuildRegistryIndexAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildRegistryIndexTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *cargoPackageType) BuildPackageIndexAsync(
	ctx context.Context,
	registry *types.Registry,
	payload types.BuildPackageIndexTaskPayload,
) error {
	err := c.cargoRegistryHelper.UpdatePackageIndex(
		ctx, payload.PrincipalID, registry.RootParentID, registry.ID, payload.Image,
	)
	if err != nil {
		return fmt.Errorf("failed to build CARGO package index for registry [%d] package [%s]: %w",
			payload.RegistryID, payload.Image, err)
	}
	return nil
}

func (c *cargoPackageType) BuildPackageMetadataAsync(
	_ context.Context,
	_ *types.Registry,
	_ types.BuildPackageMetadataTaskPayload,
) error {
	return fmt.Errorf("not implemented")
}

func (c *cargoPackageType) GetNodePathsForImage(
	_ *string,
	packageName string,
) ([]string, error) {
	return []string{"/crates/" + packageName}, nil
}

func (c *cargoPackageType) GetNodePathsForArtifact(
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

func (c *cargoPackageType) GetPkgDownloadURL(
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
