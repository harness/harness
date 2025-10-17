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

package webhook

import (
	"context"
	"fmt"
	"net/url"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/pkg"

	"github.com/rs/zerolog/log"
)

const ociPrefix = "oci://"

func GetArtifactCreatedPayload(
	ctx context.Context,
	info pkg.RegistryInfo,
	principalID int64,
	registryID int64,
	regIdentifier string,
	tag string,
	digest string,
	urlProvider urlprovider.Provider,
) registryevents.ArtifactCreatedPayload {
	payload := registryevents.ArtifactCreatedPayload{
		RegistryID:   registryID,
		PrincipalID:  principalID,
		ArtifactType: info.PackageType,
	}
	artifactURL := urlProvider.RegistryURL(ctx, info.RootIdentifier, regIdentifier) + "/" + info.Image + ":" + tag
	urlWithoutProtocol := GetRepoURLWithoutProtocol(ctx, artifactURL)
	baseArtifact := registryevents.BaseArtifact{
		Name: info.Image,
		Ref:  fmt.Sprintf("%s:%s", info.Image, tag),
	}
	//nolint:exhaustive
	switch info.PackageType {
	case artifact.PackageTypeDOCKER:
		payload.Artifact = &registryevents.DockerArtifact{
			BaseArtifact: baseArtifact,
			Tag:          tag,
			URL:          urlWithoutProtocol,
			Digest:       digest,
		}
	case artifact.PackageTypeHELM:
		payload.Artifact = &registryevents.HelmArtifact{
			BaseArtifact: baseArtifact,
			Tag:          tag,
			URL:          ociPrefix + urlWithoutProtocol,
			Digest:       digest,
		}
	}
	return payload
}

func GetArtifactDeletedPayload(
	ctx context.Context,
	principalID int64,
	registryID int64,
	regIdentifier string,
	version string,
	digest string,
	rootIdentifier string,
	packageType artifact.PackageType,
	image string,
	urlProvider urlprovider.Provider,
	isUntaggedImagesEnabled bool,
) registryevents.ArtifactDeletedPayload {
	payload := registryevents.ArtifactDeletedPayload{
		RegistryID:   registryID,
		PrincipalID:  principalID,
		ArtifactType: packageType,
	}
	var versionSeparator string
	var tag string
	if isUntaggedImagesEnabled {
		versionSeparator = "@"
	} else {
		versionSeparator = ":"
		tag = version
	}
	artifactURL := urlProvider.RegistryURL(ctx, rootIdentifier, regIdentifier) +
		"/" + image + versionSeparator + version
	urlWithoutProtocol := GetRepoURLWithoutProtocol(ctx, artifactURL)

	baseArtifact := registryevents.BaseArtifact{
		Name: image,
		Ref:  fmt.Sprintf("%s%s%s", image, versionSeparator, version),
	}
	//nolint: exhaustive
	switch packageType {
	case artifact.PackageTypeDOCKER:
		payload.Artifact = &registryevents.DockerArtifact{
			BaseArtifact: baseArtifact,
			Tag:          tag,
			Digest:       digest,
			URL:          urlWithoutProtocol,
		}
	case artifact.PackageTypeHELM:
		payload.Artifact = &registryevents.HelmArtifact{
			BaseArtifact: baseArtifact,
			Tag:          tag,
			Digest:       digest,
			URL:          ociPrefix + urlWithoutProtocol,
		}
	}
	return payload
}

func GetArtifactCreatedPayloadForCommonArtifacts(
	principalID int64,
	registryID int64,
	packageType artifact.PackageType,
	artifact string,
	version string,
) registryevents.ArtifactCreatedPayload {
	return registryevents.ArtifactCreatedPayload{
		RegistryID:   registryID,
		PrincipalID:  principalID,
		ArtifactType: packageType,
		Artifact: &registryevents.CommonArtifact{
			BaseArtifact: registryevents.BaseArtifact{
				Name: artifact,
				Ref:  fmt.Sprintf("%s:%s", artifact, version),
			},
			Version: version,
			Type:    packageType,
		},
	}
}

func GetArtifactDeletedPayloadForCommonArtifacts(
	principalID int64,
	registryID int64,
	packageType artifact.PackageType,
	artifact string,
	version string,
) registryevents.ArtifactDeletedPayload {
	return registryevents.ArtifactDeletedPayload{
		RegistryID:   registryID,
		PrincipalID:  principalID,
		ArtifactType: packageType,
		Artifact: &registryevents.CommonArtifact{
			BaseArtifact: registryevents.BaseArtifact{
				Name: artifact,
				Ref:  fmt.Sprintf("%s:%s", artifact, version),
			},
			Version: version,
			Type:    packageType,
		},
	}
}

func GetRepoURLWithoutProtocol(ctx context.Context, registryURL string) string {
	repoURL := registryURL
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msg("Error parsing URL: ")
		return ""
	}

	return parsedURL.Host + parsedURL.Path
}
