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

package gopackage

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/app/storage"
)

const (
	LatestVersionKey = "@latest"
)

type Registry interface {
	pkg.Artifact
	// Upload package to registry using harness CLI
	UploadPackage(
		ctx context.Context, info gopackagetype.ArtifactInfo,
		mod io.ReadCloser, zip io.ReadCloser,
	) (responseHeaders *commons.ResponseHeaders, err error)
	// download package file
	DownloadPackageFile(
		ctx context.Context, info gopackagetype.ArtifactInfo,
	) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error)
	// regenerate package index
	RegeneratePackageIndex(
		ctx context.Context, info gopackagetype.ArtifactInfo,
	) (*commons.ResponseHeaders, error)
	// regenerate package metadata
	RegeneratePackageMetadata(
		ctx context.Context, info gopackagetype.ArtifactInfo,
	) (*commons.ResponseHeaders, error)
}
