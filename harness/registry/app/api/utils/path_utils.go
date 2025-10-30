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

package utils

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

func GetMavenFilePath(imageName string, version string) string {
	parts := strings.SplitN(imageName, ":", 2)
	filePathPrefix := "/"
	if len(parts) == 2 {
		groupID := strings.ReplaceAll(parts[0], ".", "/")
		filePathPrefix += groupID + "/" + parts[1]
	}
	if version != "" {
		filePathPrefix += "/" + version
	}
	return filePathPrefix
}

func GetGenericFilePath(imageName string, version string) string {
	filePathPrefix := "/" + imageName
	if version != "" {
		filePathPrefix += "/" + version
	}
	return filePathPrefix
}

func GetRpmFilePath(imageName string, version string) string {
	lastDotIndex := strings.LastIndex(version, ".")
	rpmVersion := version[:lastDotIndex]
	rpmArch := version[lastDotIndex+1:]
	return "/" + imageName + "/" + rpmVersion + "/" + rpmArch
}

func GetCargoFilePath(imageName string, version string) string {
	filePathPrefix := "/crates/" + imageName
	if version != "" {
		filePathPrefix += "/" + version
	}
	return filePathPrefix
}

func GetGoFilePath(imageName string, version string) string {
	filePathPrefix := "/" + imageName + "/@v"
	if version != "" {
		filePathPrefix += "/" + version
	}
	return filePathPrefix
}

func GetHuggingFaceFilePath(imageName string, artifactType *artifact.ArtifactType, version string) string {
	filePathPrefix := "/"
	if artifactType != nil {
		filePathPrefix += string(*artifactType) + "/"
	}
	return filePathPrefix + imageName + "/" + version
}

func GetFilePath(packageType artifact.PackageType, imageName string, version string) (string, error) {
	switch packageType { //nolint:exhaustive
	case artifact.PackageTypeDOCKER:
		return "", fmt.Errorf("docker package type not supported")
	case artifact.PackageTypeHELM:
		return "", fmt.Errorf("helm package type not supported")
	case artifact.PackageTypeNPM:
		return GetGenericFilePath(imageName, version), nil
	case artifact.PackageTypeMAVEN:
		return GetMavenFilePath(imageName, version), nil
	case artifact.PackageTypePYTHON:
		return GetGenericFilePath(imageName, version), nil
	case artifact.PackageTypeGENERIC:
		return GetGenericFilePath(imageName, version), nil
	case artifact.PackageTypeNUGET:
		return GetGenericFilePath(imageName, version), nil
	case artifact.PackageTypeRPM:
		return GetRpmFilePath(imageName, version), nil
	case artifact.PackageTypeCARGO:
		return GetCargoFilePath(imageName, version), nil
	case artifact.PackageTypeGO:
		return GetGoFilePath(imageName, version), nil
	default:
		return "", fmt.Errorf("unsupported package type: %s", packageType)
	}
}

func GetFilePathWithArtifactType(packageType artifact.PackageType, imageName string, version string,
	artifactType *artifact.ArtifactType) (string, error) {
	switch packageType { //nolint:exhaustive
	case artifact.PackageTypeHUGGINGFACE:
		return GetHuggingFaceFilePath(imageName, artifactType, version), nil
	default:
		return "", fmt.Errorf("unsupported package type: %s", packageType)
	}
}
