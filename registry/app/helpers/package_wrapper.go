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

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/interfaces"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/factory"
	"github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/types"
)

type packageWrapper struct {
	packageFactory factory.PackageFactory
	regFinder      refcache.RegistryFinder
}

func NewPackageWrapper(
	packageFactory factory.PackageFactory,
	regFinder refcache.RegistryFinder,
) interfaces.PackageWrapper {
	return &packageWrapper{
		packageFactory: packageFactory,
		regFinder:      regFinder,
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

func (p *packageWrapper) GetPackageTypeFromPathPackageType(pathPackageType string) (string, error) {
	for _, pkgType := range p.packageFactory.GetAllPackageTypes() {
		pkg := p.packageFactory.Get(pkgType)
		if pkg.GetPathPackageType() == pathPackageType {
			return pkgType, nil
		}
	}
	return "", fmt.Errorf("unsupported path package type: %s", pathPackageType)
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
	if err := pkg.DeleteVersion(ctx, regInfo, imageInfo, artifactName, versionName); err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}
	if err := p.ReportDeleteVersionEvent(ctx, regInfo.RegistryID, artifactName, versionName); err != nil {
		return fmt.Errorf("failed to report delete version event: %w", err)
	}
	if err := p.ReportBuildPackageIndexEvent(ctx, regInfo.RegistryID, artifactName); err != nil {
		return fmt.Errorf("failed to report build package index event: %w", err)
	}
	if err := p.ReportBuildRegistryIndexEvent(ctx, regInfo.RegistryID, make([]types.SourceRef, 0)); err != nil {
		return fmt.Errorf("failed to report build registry index event: %w", err)
	}
	return nil
}

func (p *packageWrapper) ReportDeleteVersionEvent(
	ctx context.Context,
	registryID int64,
	artifactName string,
	versionName string,
) error {
	session, ok := request.AuthSessionFrom(ctx)
	if !ok {
		return fmt.Errorf("failed to get auth session")
	}
	registry, err := p.regFinder.FindByID(ctx, registryID)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	pkg := p.GetPackage(string(registry.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", registry.PackageType)
	}
	pkg.ReportDeleteVersionEvent(ctx, session.Principal.ID, registry.ID, artifactName, versionName)
	return nil
}

func (p *packageWrapper) ReportBuildPackageIndexEvent(
	ctx context.Context,
	registryID int64,
	artifactName string,
) error {
	registry, err := p.regFinder.FindByID(ctx, registryID)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	pkg := p.GetPackage(string(registry.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", registry.PackageType)
	}
	pkg.ReportBuildPackageIndexEvent(ctx, registry.ID, artifactName)
	return nil
}

func (p *packageWrapper) ReportBuildRegistryIndexEvent(
	ctx context.Context,
	registryID int64,
	sourceRefs []types.SourceRef,
) error {
	registry, err := p.regFinder.FindByID(ctx, registryID)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	pkg := p.GetPackage(string(registry.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", registry.PackageType)
	}
	pkg.ReportBuildRegistryIndexEvent(ctx, registry.ID, sourceRefs)
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
	if err := pkg.DeleteArtifact(ctx, regInfo, artifactName); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}
	if err := p.ReportBuildRegistryIndexEvent(ctx, regInfo.RegistryID, make([]types.SourceRef, 0)); err != nil {
		return fmt.Errorf("failed to report build registry index event: %w", err)
	}
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

func (p *packageWrapper) BuildRegistryIndexAsync(
	ctx context.Context,
	payload types.BuildRegistryIndexTaskPayload,
) error {
	registry, err := p.regFinder.FindByID(ctx, payload.RegistryID)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	pkg := p.GetPackage(string(registry.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", registry.PackageType)
	}
	return pkg.BuildRegistryIndexAsync(ctx, registry, payload)
}

func (p *packageWrapper) BuildPackageIndexAsync(
	ctx context.Context,
	payload types.BuildPackageIndexTaskPayload,
) error {
	registry, err := p.regFinder.FindByID(ctx, payload.RegistryID)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	pkg := p.GetPackage(string(registry.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", registry.PackageType)
	}
	return pkg.BuildPackageIndexAsync(ctx, registry, payload)
}

func (p *packageWrapper) BuildPackageMetadataAsync(
	ctx context.Context,
	payload types.BuildPackageMetadataTaskPayload,
) error {
	registry, err := p.regFinder.FindByID(ctx, payload.RegistryID)
	if err != nil {
		return fmt.Errorf("failed to find registry: %w", err)
	}
	pkg := p.GetPackage(string(registry.PackageType))
	if pkg == nil {
		return fmt.Errorf("unsupported package type: %s", registry.PackageType)
	}
	return pkg.BuildPackageMetadataAsync(ctx, registry, payload)
}

func (p *packageWrapper) GetNodePathsForImage(
	packageType string,
	artifactType *string,
	packageName string,
) []string {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return []string{}
	}
	return pkg.GetNodePathsForImage(artifactType, packageName)
}

func (p *packageWrapper) GetNodePathsForArtifact(
	packageType string,
	artifactType *string,
	packageName string,
	version string,
) []string {
	pkg := p.GetPackage(packageType)
	if pkg == nil {
		return []string{}
	}
	return pkg.GetNodePathsForArtifact(artifactType, packageName, version)
}
