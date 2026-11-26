//  Copyright 2026 Harness, Inc.
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

package metadata

import (
	"context"
	"errors"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const rmdRegistryBase = "https://reg.example.com"

// rmdURLProvider is a URL provider fake that only supports RegistryURL.
type rmdURLProvider struct {
	urlprovider.Provider
}

func (rmdURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, params ...string) string {
	return strings.Join(append([]string{rmdRegistryBase, "pkg", regRef, pkgType}, params...), "/")
}

func (rmdURLProvider) RegistryURL(_ context.Context, params ...string) string {
	return strings.Join(append([]string{rmdRegistryBase}, params...), "/")
}

// rmdPkgWrapper is a package wrapper fake with overridable behaviour.
type rmdPkgWrapper struct {
	interfaces.PackageWrapper
	artifactMetadata *artifact.ArtifactMetadata
	versionMetadata  *artifact.ArtifactVersionMetadata
	fileMetadata     *artifact.FileDetail
	invalidPkgType   bool
	invalidRepoType  bool
	urlRequired      bool
	invalidSource    bool
	fileOpsSupported bool
	filePath         string
	filePathErr      error
	deleteVersionErr error
	deleteErr        error
	pkgTypeFromPath  string
	pkgTypeErr       error
	pkgURL           string
	pkgURLErr        error
	detail           *artifact.ArtifactDetail
	detailErr        error
}

func (w rmdPkgWrapper) GetArtifactDetail(
	_ string, _ *types.Image, _ *types.Artifact, _ int64,
) (*artifact.ArtifactDetail, error) {
	return w.detail, w.detailErr
}

func (w rmdPkgWrapper) GetPackageURL(_ context.Context, _ string, _ string, _ string) (string, error) {
	return w.pkgURL, w.pkgURLErr
}

func (w rmdPkgWrapper) IsValidPackageType(_ string) bool { return !w.invalidPkgType }

func (w rmdPkgWrapper) IsValidPackageTypes(_ []string) bool { return !w.invalidPkgType }

func (w rmdPkgWrapper) IsValidRepoType(_ string) bool { return !w.invalidRepoType }

func (w rmdPkgWrapper) ValidateRepoType(_ string, _ string) bool { return !w.invalidRepoType }

func (w rmdPkgWrapper) IsURLRequiredForUpstreamSource(_ string, _ string) bool { return w.urlRequired }

func (w rmdPkgWrapper) ValidateUpstreamSource(_ string, _ string) bool { return !w.invalidSource }

func (w rmdPkgWrapper) IsFileOperationSupported(_ string) bool { return w.fileOpsSupported }

func (w rmdPkgWrapper) GetFilePath(_ string, _ string, _ string) (string, error) {
	return w.filePath, w.filePathErr
}

func (w rmdPkgWrapper) GetPackageTypeFromPathPackageType(_ string) (string, error) {
	return w.pkgTypeFromPath, w.pkgTypeErr
}

func (w rmdPkgWrapper) DeleteArtifactVersion(
	_ context.Context, _ *types.RegistryRequestBaseInfo, _ *types.Image, _ string, _ string,
) error {
	return w.deleteVersionErr
}

func (w rmdPkgWrapper) DeleteArtifact(_ context.Context, _ *types.RegistryRequestBaseInfo, _ string) error {
	return w.deleteErr
}

func (w rmdPkgWrapper) ReportBuildRegistryIndexEvent(
	_ context.Context, _ int64, _ []types.SourceRef,
) error {
	return nil
}

func (w rmdPkgWrapper) GetArtifactMetadata(_ types.ArtifactMetadata) *artifact.ArtifactMetadata {
	return w.artifactMetadata
}

func (w rmdPkgWrapper) GetArtifactVersionMetadata(
	_ string, _ string, _ types.NonOCIArtifactMetadata,
) *artifact.ArtifactVersionMetadata {
	if w.versionMetadata == nil {
		return nil
	}
	cp := *w.versionMetadata
	return &cp
}

func (w rmdPkgWrapper) GetFileMetadata(
	_ context.Context, _ string, _ string, _ string, _ string, _ string, _ types.FileNodeMetadata,
) *artifact.FileDetail {
	return w.fileMetadata
}

// rmdSpaceFinder is a space finder fake.
type rmdSpaceFinder struct {
	space *coretypes.SpaceCore
	err   error
}

func (f rmdSpaceFinder) FindByID(_ context.Context, _ int64) (*coretypes.SpaceCore, error) {
	return f.space, f.err
}

func (f rmdSpaceFinder) FindByRef(_ context.Context, _ string) (*coretypes.SpaceCore, error) {
	return f.space, f.err
}

// rmdAuthorizer is an authorizer fake.
type rmdAuthorizer struct {
	authz.Authorizer
	allow bool
	err   error
	// denySecret rejects only secret access checks.
	denySecret bool
}

func (a rmdAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *coretypes.Scope, resource *coretypes.Resource, _ enum.Permission,
) (bool, error) {
	if a.denySecret && resource != nil && resource.Type == enum.ResourceTypeSecret {
		return false, nil
	}
	return a.allow, a.err
}

func (a rmdAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, _ ...coretypes.PermissionCheck,
) (bool, error) {
	return a.allow, a.err
}

// rmdHelper is a registry metadata helper fake.
type rmdHelper struct {
	interfaces.RegistryMetadataHelper
	info         *types.RegistryRequestBaseInfo
	infoErr      error
	secretSpace  int64
	secretSpaceE error
	core         *coretypes.WebhookCore
	coreErr      error
	resp         *artifact.Webhook
	respErr      error
}

func (h rmdHelper) MapToWebhookCore(
	_ context.Context, _ artifact.WebhookRequest, _ *types.RegistryRequestBaseInfo,
) (*coretypes.WebhookCore, error) {
	if h.coreErr != nil {
		return nil, h.coreErr
	}
	if h.core == nil {
		return &coretypes.WebhookCore{}, nil
	}
	cp := *h.core
	return &cp, nil
}

func (h rmdHelper) MapToWebhookResponseEntity(
	_ context.Context, w *coretypes.WebhookCore,
) (*artifact.Webhook, error) {
	if h.respErr != nil {
		return nil, h.respErr
	}
	if h.resp != nil {
		return h.resp, nil
	}
	return &artifact.Webhook{Identifier: w.Identifier}, nil
}

func (h rmdHelper) GetSecretSpaceID(_ context.Context, _ *string) (int64, error) {
	return h.secretSpace, h.secretSpaceE
}

func (h rmdHelper) GetRegistryRequestBaseInfo(
	_ context.Context, _ string, _ string,
) (*types.RegistryRequestBaseInfo, error) {
	return h.info, h.infoErr
}

func (h rmdHelper) GetPermissionChecks(
	_ *coretypes.SpaceCore, _ string, _ enum.Permission,
) []coretypes.PermissionCheck {
	return nil
}

func rmdAdminCtx() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{
		Principal: coretypes.Principal{ID: 7, Type: enum.PrincipalTypeUser, Email: "dev@example.com"},
		Metadata:  &auth.EmptyMetadata{},
	})
}

func rmdAnonCtx() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{
		Principal: auth.AnonymousPrincipal,
		Metadata:  &auth.EmptyMetadata{},
	})
}

func rmdErrUnauthorized() error { return apiauth.ErrUnauthorized }

func rmdErrForbidden() error { return apiauth.ErrForbidden }

var errRmdNotAvailable = errors.New("not available")
