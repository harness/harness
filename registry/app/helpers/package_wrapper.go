//  Copyright 2023 Harness, Inc.
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

package helpers

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/api/interfaces"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/factory"
	"github.com/harness/gitness/registry/types"
)

type packageWrapper struct {
	packageFactory factory.PackageFactory
}

func NewPackageWrapper(
	packageFactory factory.PackageFactory,
) interfaces.PackageWrapper {
	return &packageWrapper{
		packageFactory: packageFactory,
	}
}

func (p *packageWrapper) GetPackage(packageType string) interfaces.PackageHelper {
	return p.packageFactory.Get(packageType)
}

func (p *packageWrapper) IsValidPackageType(packageType string) bool {
	if packageType == "" {
		return true
	}
	return p.packageFactory.IsValidPackageType(packageType)
}

func (p *packageWrapper) IsValidPackageTypes(packageTypes []string) bool {
	for _, packageType := range packageTypes {
		if !p.IsValidPackageType(packageType) {
			return false
		}
	}
	return true
}

func (p *packageWrapper) IsValidRepoType(repoType string) bool {
	if repoType == "" {
		return true
	}
	for _, pkgType := range p.packageFactory.GetAllPackageTypes() {
		pkg := p.packageFactory.Get(pkgType)
		if pkg.IsValidRepoType(repoType) {
			return true
		}
	}
	return false
}

func (p *packageWrapper) IsValidRepoTypes(repoTypes []string) bool {
	for _, repoType := range repoTypes {
		if !p.IsValidRepoType(repoType) {
			return false
		}
	}
	return true
}

func (p *packageWrapper) ValidateRepoType(packageType string, repoType string) bool {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return false
	}
	return pkg.IsValidRepoType(repoType)
}

func (p *packageWrapper) IsValidUpstreamSource(upstreamSource string) bool {
	if upstreamSource == "" {
		return true
	}
	for _, pkgType := range p.packageFactory.GetAllPackageTypes() {
		pkg := p.packageFactory.Get(pkgType)
		if pkg.IsValidUpstreamSource(upstreamSource) {
			return true
		}
	}
	return false
}

func (p *packageWrapper) IsValidUpstreamSources(upstreamSources []string) bool {
	for _, upstreamSource := range upstreamSources {
		if !p.IsValidUpstreamSource(upstreamSource) {
			return false
		}
	}
	return true
}

func (p *packageWrapper) ValidateUpstreamSource(packageType string, upstreamSource string) bool {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return false
	}
	return pkg.IsValidUpstreamSource(upstreamSource)
}

func (p *packageWrapper) IsURLRequiredForUpstreamSource(packageType string, upstreamSource string) bool {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return false
	}
	return pkg.IsURLRequiredForUpstreamSource(upstreamSource)
}

func (p *packageWrapper) DeleteArtifactVersion(
	ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	imageInfo *types.Image,
	artifactName string,
	versionName string,
) error {
	pkg := p.GetPackage(string(regInfo.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", regInfo.PackageType)
	}
	err := pkg.DeleteVersion(ctx, regInfo, imageInfo, artifactName, versionName)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}
	pkg.ReportDeleteVersionEvent(ctx, regInfo.RegistryID, imageInfo.ID, artifactName, versionName)
	pkg.ReportBuildPackageIndexEvent(ctx, regInfo.RegistryID, artifactName)
	pkg.ReportBuildRegistryIndexEvent(ctx, regInfo.RegistryID, make([]types.SourceRef, 0))
	return nil
}

func (p *packageWrapper) DeleteArtifact(
	ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	artifactName string,
) error {
	pkg := p.GetPackage(string(regInfo.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", regInfo.PackageType)
	}
	err := pkg.DeleteArtifact(ctx, regInfo, artifactName)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}
	pkg.ReportBuildRegistryIndexEvent(ctx, regInfo.RegistryID, make([]types.SourceRef, 0))
	return nil
}

func (p *packageWrapper) GetFilePath(
	packageType string,
	artifactName string,
	versionName string,
) (string, error) {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return "", fmt.Errorf("unsupported package type: %s", packageType)
	}
	return pkg.GetFilePath(artifactName, versionName), nil
}

func (p *packageWrapper) GetPackageURL(
	ctx context.Context,
	rootIdentifier string,
	registryIdentifier string,
	packageType string,
) (string, error) {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return "", fmt.Errorf("unsupported package type: %s", packageType)
	}
	return pkg.GetPackageURL(ctx, rootIdentifier, registryIdentifier), nil
}

func (p *packageWrapper) GetArtifactMetadata(
	artifact types.ArtifactMetadata,
) *artifactapi.ArtifactMetadata {
	pkg := p.GetPackage(string(artifact.PackageType))
	if pkg == nil {
		return nil
	}
	return pkg.GetArtifactMetadata(artifact)
}

func (p *packageWrapper) GetArtifactVersionMetadata(
	packageType string,
	image string,
	tag types.NonOCIArtifactMetadata,
) *artifactapi.ArtifactVersionMetadata {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return nil
	}
	return pkg.GetArtifactVersionMetadata(image, tag)
}

func (p *packageWrapper) GetFileMetadata(
	ctx context.Context,
	rootIdentifier string,
	registryIdentifier string,
	packageType string,
	artifactName string,
	version string,
	file types.FileNodeMetadata,
) *artifactapi.FileDetail {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return nil
	}
	return pkg.GetFileMetadata(
		ctx,
		rootIdentifier,
		registryIdentifier,
		artifactName,
		version,
		file,
	)
}

func (p *packageWrapper) GetArtifactDetail(
	packageType string,
	img *types.Image,
	art *types.Artifact,
	downloadCount int64,
) (*artifactapi.ArtifactDetail, error) {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return nil, fmt.Errorf("unsupported package type: %s", packageType)
	}
	return pkg.GetArtifactDetail(img, art, downloadCount)
}

func (p *packageWrapper) GetClientSetupDetails(
	ctx context.Context,
	regRef string,
	image *artifactapi.ArtifactParam,
	tag *artifactapi.VersionParam,
	registryType artifactapi.RegistryType,
	packageType string,
) (*artifactapi.ClientSetupDetails, error) {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return nil, fmt.Errorf("unsupported package type: %s", packageType)
	}
	return pkg.GetClientSetupDetails(ctx, regRef, image, tag, registryType)
}
