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
	artifactName := strings.ReplaceAll(imageName, ".", "/")
	artifactName = strings.ReplaceAll(artifactName, ":", "/")
	filePathPrefix := "/" + artifactName
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

func GetFilePath(
	packageType artifact.PackageType,
	imageName string, version string,
) (string, error) {
	switch packageType {
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
		return "", fmt.Errorf("nuget package type not supported")
	case artifact.PackageTypeRPM:
		return GetRpmFilePath(imageName, version), nil
	default:
		return "", fmt.Errorf("unsupported package type: %s", packageType)
	}
}
