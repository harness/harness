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
	"net/http"
)

// PackageArtifactInfo is an interface that must be implemented by all package-specific
// artifact info types. It ensures that all package artifact infos can be converted to
// the base ArtifactInfo type.
type PackageArtifactInfo interface {
	BaseArtifactInfo() ArtifactInfo
	GetImageVersion() (bool, string)

	GetVersion() string

	GetFileName() string
}

// ArtifactInfoProvider is an interface that must be implemented by package handlers
// to provide artifact information from HTTP requests.
type ArtifactInfoProvider interface {
	// GetPackageArtifactInfo returns package-specific artifact info that implements
	// the PackageArtifactInfo interface
	GetPackageArtifactInfo(r *http.Request) (PackageArtifactInfo, error)
}
