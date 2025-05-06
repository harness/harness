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

package nuget

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
)

type Registry interface {
	pkg.Artifact

	UploadPackage(
		ctx context.Context,
		info nuget.ArtifactInfo,
		fileReader io.ReadCloser,
	) (*commons.ResponseHeaders, string, error)

	DownloadPackage(ctx context.Context, info nuget.ArtifactInfo) (
		*commons.ResponseHeaders,
		*storage.FileReader,
		string,
		error,
	)

	ListPackageVersion(ctx context.Context, info nuget.ArtifactInfo) (*nuget.PackageVersion, error)

	GetPackageMetadata(ctx context.Context, info nuget.ArtifactInfo) (*nuget.RegistrationIndexResponse, error)

	GetPackageVersionMetadata(ctx context.Context, info nuget.ArtifactInfo) (*nuget.RegistrationLeafResponse, error)

	GetServiceEndpoint(ctx context.Context, info nuget.ArtifactInfo) *nuget.ServiceEndpoint
}
