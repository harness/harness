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

package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/common"
	"github.com/harness/gitness/registry/utils"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetClientSetupDetails(
	ctx context.Context,
	r artifact.GetClientSetupDetailsRequestObject,
) (artifact.GetClientSetupDetailsResponseObject, error) {
	regRefParam := r.RegistryRef
	imageParam := r.Params.Artifact
	versionParam := r.Params.Version

	regInfo, _ := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(regRefParam))

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetClientSetupDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetClientSetupDetails403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	reg, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)
	if err != nil {
		return artifact.GetClientSetupDetails404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this ref"),
			),
		}, err
	}

	packageType := string(reg.PackageType)

	response := c.GenerateClientSetupDetails(
		ctx, packageType, imageParam, versionParam, regInfo.RegistryRef,
		regInfo.RegistryType,
	)

	if response == nil {
		return artifact.GetClientSetupDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, "Package type not supported"),
			),
		}, nil
	}
	return artifact.GetClientSetupDetails200JSONResponse{
		ClientSetupDetailsResponseJSONResponse: *response,
	}, nil
}

func (c *APIController) GenerateClientSetupDetails(
	ctx context.Context,
	packageType string,
	image *artifact.ArtifactParam,
	version *artifact.VersionParam,
	registryRef string,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	blankString := ""
	switch packageType {
	case string(artifact.PackageTypeRPM):
		return c.generateRpmClientSetupDetail(ctx, image, version, registryRef)
	case string(artifact.PackageTypeMAVEN):
		return c.generateMavenClientSetupDetail(ctx, image, version, registryRef, registryType)
	case string(artifact.PackageTypeHELM):
		return c.generateHelmClientSetupDetail(ctx, blankString, registryRef, image, version, registryType)
	case string(artifact.PackageTypeGENERIC):
		return c.generateGenericClientSetupDetail(ctx, blankString, registryRef, image, version, registryType)
	case string(artifact.PackageTypePYTHON):
		return c.generatePythonClientSetupDetail(ctx, registryRef, image, version, registryType)
	case string(artifact.PackageTypeNPM):
		return c.generateNpmClientSetupDetail(ctx, registryRef, image, version, registryType)
	case string(artifact.PackageTypeDOCKER):
		return c.generateDockerClientSetupDetail(ctx, blankString, registryType, registryRef, image, version)
	case string(artifact.PackageTypeNUGET):
		return c.generateNugetClientSetupDetail(ctx, registryRef, image, version, registryType)
	case string(artifact.PackageTypeGO):
		return c.generateGoClientSetupDetail(ctx, registryRef, image, version, registryType)
	case string(artifact.PackageTypeHUGGINGFACE):
		return c.generateHuggingFaceClientSetupDetail(ctx, registryRef, image, version, registryType)
	default:
		clientSetupDetails, err := c.PackageWrapper.GetClientSetupDetails(
			ctx, registryRef, image, version, registryType, packageType,
		)
		if err != nil {
			log.Error().Ctx(ctx).Err(err).Msg("Failed to get client setup details")
			return nil
		}
		return &artifact.ClientSetupDetailsResponseJSONResponse{
			Data:   *clientSetupDetails,
			Status: artifact.StatusSUCCESS,
		}
	}
}

func (c *APIController) generateDockerClientSetupDetail(
	ctx context.Context,
	blankString string,
	registryType artifact.RegistryType,
	registryRef string,
	image *artifact.ArtifactParam,
	version *artifact.VersionParam,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousDockerClientSetupDetails(ctx, blankString, registryType)
	} else {
		clientSetupDetails = c.getDockerClientSetupDetails(ctx, blankString, registryType)
	}

	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, version, "", "", "")

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousDockerClientSetupDetails(
	ctx context.Context,
	blankString string,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	header2 := "Pull an image"
	section2step1Header := "Run this Docker command in your terminal to pull image."
	var dockerPullValue string
	if c.UntaggedImagesEnabled(ctx) {
		dockerPullValue = "docker pull <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>@<DIGEST>"
	} else {
		dockerPullValue = "docker pull <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<TAG>"
	}
	section2step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &dockerPullValue},
	}
	section2step1Type := artifact.ClientSetupStepTypeStatic
	section2Steps := []artifact.ClientSetupStep{
		{
			Header:   &section2step1Header,
			Commands: &section2step1Commands,
			Type:     &section2step1Type,
		},
	}
	section2 := artifact.ClientSetupSection{
		Header: &header2,
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section2Steps,
	})
	sections := []artifact.ClientSetupSection{
		section2,
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section2,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Docker Client Setup",
		SecHeader:  "Follow these instructions to install/use Docker artifacts or compatible packages.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getDockerClientSetupDetails(
	ctx context.Context,
	blankString string,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	loginUsernameLabel := "Username: <USERNAME>"
	loginUsernameValue := "<USERNAME>"
	loginPasswordLabel := "Password: *see step 2*"

	header1 := "Login to Docker"
	section1step1Header := "Run this Docker command in your terminal to authenticate the client."
	dockerLoginValue := "docker login <LOGIN_HOSTNAME>"
	section1step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &dockerLoginValue},
		{Label: &loginUsernameLabel, Value: &loginUsernameValue},
		{Label: &loginPasswordLabel, Value: &blankString},
	}
	section1step1Type := artifact.ClientSetupStepTypeStatic
	section1step2Header := "For the Password field above, generate an identity token"
	section1step2Type := artifact.ClientSetupStepTypeGenerateToken
	section1Steps := []artifact.ClientSetupStep{
		{
			Header:   &section1step1Header,
			Commands: &section1step1Commands,
			Type:     &section1step1Type,
		},
		{
			Header: &section1step2Header,
			Type:   &section1step2Type,
		},
	}
	section1 := artifact.ClientSetupSection{
		Header: &header1,
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section1Steps,
	})
	header2 := "Pull an image"
	section2step1Header := "Run this Docker command in your terminal to pull image."
	var dockerPullValue string
	if c.UntaggedImagesEnabled(ctx) {
		dockerPullValue = "docker pull <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>@<DIGEST>"
	} else {
		dockerPullValue = "docker pull <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<TAG>"
	}
	section2step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &dockerPullValue},
	}
	section2step1Type := artifact.ClientSetupStepTypeStatic
	section2Steps := []artifact.ClientSetupStep{
		{
			Header:   &section2step1Header,
			Commands: &section2step1Commands,
			Type:     &section2step1Type,
		},
	}
	section2 := artifact.ClientSetupSection{
		Header: &header2,
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section2Steps,
	})
	header3 := "Retag and Push the image"
	section3step1Header := "Run this Docker command in your terminal to tag the image."
	var dockerTagValue string
	if c.UntaggedImagesEnabled(ctx) {
		dockerTagValue = "docker tag <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>@<DIGEST> " +
			"<HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<NEW_TAG>"
	} else {
		dockerTagValue = "docker tag <IMAGE_NAME>:<TAG> <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<TAG>"
	}
	section3step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &dockerTagValue},
	}
	section3step1Type := artifact.ClientSetupStepTypeStatic
	section3step2Header := "Run this Docker command in your terminal to push the image."
	dockerPushValue := "docker push <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<NEW_TAG>"
	section3step2Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &dockerPushValue},
	}
	section3step2Type := artifact.ClientSetupStepTypeStatic
	section3Steps := []artifact.ClientSetupStep{
		{
			Header:   &section3step1Header,
			Commands: &section3step1Commands,
			Type:     &section3step1Type,
		},
		{
			Header:   &section3step2Header,
			Commands: &section3step2Commands,
			Type:     &section3step2Type,
		},
	}
	section3 := artifact.ClientSetupSection{
		Header: &header3,
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section3Steps,
	})
	sections := []artifact.ClientSetupSection{
		section1,
		section2,
		section3,
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section1,
			section2,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Docker Client Setup",
		SecHeader:  "Follow these instructions to install/use Docker artifacts or compatible packages.",
		Sections:   sections,
	}
	return clientSetupDetails
}

//nolint:lll
func (c *APIController) generateGenericClientSetupDetail(
	ctx context.Context,
	blankString string,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	session, _ := request.AuthSessionFrom(ctx)
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousGenericClientSetupDetails(blankString, registryType)
	} else {
		clientSetupDetails = c.getGenericClientSetupDetails(blankString, registryType)
	}
	//nolint:lll
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, "", registryRef, image, tag, "", "",
		string(artifact.PackageTypeGENERIC))
	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousGenericClientSetupDetails(
	blankString string,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	header3 := "Download Artifact"
	//nolint:lll
	section3step1Header := "Run this command in your terminal to download the artifact at package level. This command works for non-nested paths."
	//nolint:lll
	pullValue := "curl --location '<HOSTNAME>/generic/<ARTIFACT_NAME>/<VERSION>?filename=<FILENAME>' " +
		"-J -O"
	section3step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &pullValue},
	}
	section3step1Type := artifact.ClientSetupStepTypeStatic
	section3steps := []artifact.ClientSetupStep{
		{
			Header:   &section3step1Header,
			Commands: &section3step1Commands,
			Type:     &section3step1Type,
		},
	}
	section3 := artifact.ClientSetupSection{
		Header: &header3,
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section3steps,
	})

	header4 := "File Operations (Supports both nested and flat paths)"

	section4step2Header := "Download a file from a specific path within the package."
	//nolint:lll
	fileDownloadValue := "curl --location '<HOSTNAME>/files/<ARTIFACT_NAME>/<VERSION>/<NESTED_FILE_PATH>' -J -O"
	section4step2Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &fileDownloadValue},
	}

	section4step3Header := "Get file metadata from a specific path (HEAD request)."
	//nolint:lll
	fileHeadValue := "curl --location --head '<HOSTNAME>/files/<ARTIFACT_NAME>/<VERSION>/<NESTED_FILE_PATH>'"
	section4step3Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &fileHeadValue},
	}

	section4step1Type := artifact.ClientSetupStepTypeStatic
	section4steps := []artifact.ClientSetupStep{
		{
			Header:   &section4step2Header,
			Commands: &section4step2Commands,
			Type:     &section4step1Type,
		},
		{
			Header:   &section4step3Header,
			Commands: &section4step3Commands,
			Type:     &section4step1Type,
		},
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		section4steps = []artifact.ClientSetupStep{
			{
				Header:   &section4step2Header,
				Commands: &section4step2Commands,
				Type:     &section4step1Type,
			},
			{
				Header:   &section4step3Header,
				Commands: &section4step3Commands,
				Type:     &section4step1Type,
			},
		}
	}
	section4 := artifact.ClientSetupSection{
		Header: &header4,
	}
	_ = section4.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section4steps,
	})

	sections := []artifact.ClientSetupSection{
		section3,
		section4,
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section3,
			section4,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Generic Client Setup",
		SecHeader:  "Follow these instructions to install/use Generic artifacts or compatible packages.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getGenericClientSetupDetails(
	blankString string,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	header1 := "Generate identity token"
	section1Header := "An identity token will serve as the password for uploading and downloading artifact."
	section1Type := artifact.ClientSetupStepTypeGenerateToken
	section1steps := []artifact.ClientSetupStep{
		{
			Header: &section1Header,
			Type:   &section1Type,
		},
	}
	section1 := artifact.ClientSetupSection{
		Header: &header1,
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section1steps,
	})

	header2 := "Upload Artifact"
	//nolint:lll
	section2step1Header := "Run this curl command in your terminal to push the artifact at package level. This command works for non-nested paths."
	//nolint:lll
	pushValue := "curl --location --request PUT '<HOSTNAME>/generic/<ARTIFACT_NAME>/<VERSION>' \\\n--form 'filename=\"<FILENAME>\"' \\\n--form 'file=@\"<FILE_PATH>\"' \\\n--form 'description=\"<DESC>\"' \\\n--header '<AUTH_HEADER_PREFIX> <API_KEY>'"
	section2step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &pushValue},
	}
	section2step1Type := artifact.ClientSetupStepTypeStatic
	section2steps := []artifact.ClientSetupStep{
		{
			Header:   &section2step1Header,
			Commands: &section2step1Commands,
			Type:     &section2step1Type,
		},
	}
	section2 := artifact.ClientSetupSection{
		Header: &header2,
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section2steps,
	})

	header3 := "Download Artifact"
	//nolint:lll
	section3step1Header := "Run this command in your terminal to download the artifact at package level. This command works for non-nested paths."
	//nolint:lll
	pullValue := "curl --location '<HOSTNAME>/generic/<ARTIFACT_NAME>/<VERSION>?filename=<FILENAME>' \\\n --header '<AUTH_HEADER_PREFIX> <API_KEY>' " +
		"-J -O"
	section3step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &pullValue},
	}
	section3step1Type := artifact.ClientSetupStepTypeStatic
	section3steps := []artifact.ClientSetupStep{
		{
			Header:   &section3step1Header,
			Commands: &section3step1Commands,
			Type:     &section3step1Type,
		},
	}
	section3 := artifact.ClientSetupSection{
		Header: &header3,
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section3steps,
	})

	header4 := "File Operations (Supports both nested and flat paths)"
	section4step1Header := "Upload a file to a specific path within the package."
	//nolint:lll
	fileUploadValue := "curl -XPUT '<HOSTNAME>/files/<ARTIFACT_NAME>/<VERSION>/<NESTED_FILE_PATH>' \\\n--header '<AUTH_HEADER_PREFIX> <API_KEY>' -T '<FILE_PATH>'"
	section4step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &fileUploadValue},
	}

	section4step2Header := "Download a file from a specific path within the package."
	//nolint:lll
	fileDownloadValue := "curl --location '<HOSTNAME>/files/<ARTIFACT_NAME>/<VERSION>/<NESTED_FILE_PATH>' --header '<AUTH_HEADER_PREFIX> <API_KEY>' -J -O"
	section4step2Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &fileDownloadValue},
	}

	section4step3Header := "Get file metadata from a specific path (HEAD request)."
	//nolint:lll
	fileHeadValue := "curl --location --head '<HOSTNAME>/files/<ARTIFACT_NAME>/<VERSION>/<NESTED_FILE_PATH>' --header '<AUTH_HEADER_PREFIX> <API_KEY>'"
	section4step3Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &fileHeadValue},
	}

	section4step4Header := "Delete a file from a specific path within the package."
	//nolint:lll
	fileDeleteValue := "curl --location --request DELETE '<HOSTNAME>/files/<ARTIFACT_NAME>/<VERSION>/<NESTED_FILE_PATH>' --header '<AUTH_HEADER_PREFIX> <API_KEY>'"
	section4step4Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &fileDeleteValue},
	}

	section4step4UpstreamHeader := "Delete a file from a specific path within the package (only deletes cached file)."

	section4step1Type := artifact.ClientSetupStepTypeStatic
	section4steps := []artifact.ClientSetupStep{
		{
			Header:   &section4step1Header,
			Commands: &section4step1Commands,
			Type:     &section4step1Type,
		},
		{
			Header:   &section4step2Header,
			Commands: &section4step2Commands,
			Type:     &section4step1Type,
		},
		{
			Header:   &section4step3Header,
			Commands: &section4step3Commands,
			Type:     &section4step1Type,
		},
		{
			Header:   &section4step4Header,
			Commands: &section4step4Commands,
			Type:     &section4step1Type,
		},
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		section4steps = []artifact.ClientSetupStep{
			{
				Header:   &section4step2Header,
				Commands: &section4step2Commands,
				Type:     &section4step1Type,
			},
			{
				Header:   &section4step3Header,
				Commands: &section4step3Commands,
				Type:     &section4step1Type,
			},
			{
				Header:   &section4step4UpstreamHeader,
				Commands: &section4step4Commands,
				Type:     &section4step1Type,
			},
		}
	}
	section4 := artifact.ClientSetupSection{
		Header: &header4,
	}
	_ = section4.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section4steps,
	})

	sections := []artifact.ClientSetupSection{
		section1,
		section2,
		section3,
		section4,
	}

	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section1,
			section3,
			section4,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Generic Client Setup",
		SecHeader:  "Follow these instructions to install/use Generic artifacts or compatible packages.",
		Sections:   sections,
	}
	return clientSetupDetails
}

//nolint:lll
func (c *APIController) generateHelmClientSetupDetail(
	ctx context.Context,
	blankString string,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	loginUsernameLabel := "Username: <USERNAME>"
	loginUsernameValue := "<USERNAME>"
	loginPasswordLabel := "Password: *see step 2*"
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousHelmClientSetupDetails(ctx, blankString, registryType)
	} else {
		clientSetupDetails = c.getHelmClientSetupDetails(ctx, blankString,
			loginUsernameLabel, loginUsernameValue, loginPasswordLabel, registryType)
	}

	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username,
		registryRef, image, tag, "", "", "")
	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousHelmClientSetupDetails(
	ctx context.Context,
	blankString string,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	header3 := "Pull a version"
	section3step1Header := "Run this Helm command in your terminal to pull a specific chart version."
	var helmPullValue string
	if c.UntaggedImagesEnabled(ctx) {
		helmPullValue = "helm pull oci://<HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>@<DIGEST>"
	} else {
		helmPullValue = "helm pull oci://<HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME> --version <TAG>"
	}
	section3step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &helmPullValue},
	}
	section3step1Type := artifact.ClientSetupStepTypeStatic
	section3Steps := []artifact.ClientSetupStep{
		{
			Header:   &section3step1Header,
			Commands: &section3step1Commands,
			Type:     &section3step1Type,
		},
	}
	section3 := artifact.ClientSetupSection{
		Header: &header3,
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section3Steps,
	})

	sections := []artifact.ClientSetupSection{
		section3,
	}
	if registryType == artifact.RegistryTypeUPSTREAM {
		sections = []artifact.ClientSetupSection{
			section3,
		}
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Helm Client Setup",
		SecHeader:  "Follow these instructions to install/use Helm artifacts or compatible packages.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getHelmClientSetupDetails(
	ctx context.Context,
	blankString string,
	loginUsernameLabel string,
	loginUsernameValue string,
	loginPasswordLabel string,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	header1 := "Login to Helm"
	section1step1Header := "Run this Helm command in your terminal to authenticate the client."
	helmLoginValue := "helm registry login <LOGIN_HOSTNAME>"
	section1step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &helmLoginValue},
		{Label: &loginUsernameLabel, Value: &loginUsernameValue},
		{Label: &loginPasswordLabel, Value: &blankString},
	}
	section1step1Type := artifact.ClientSetupStepTypeStatic
	section1step2Header := "For the Password field above, generate an identity token"
	section1step2Type := artifact.ClientSetupStepTypeGenerateToken
	section1Steps := []artifact.ClientSetupStep{
		{
			Header:   &section1step1Header,
			Commands: &section1step1Commands,
			Type:     &section1step1Type,
		},
		{
			Header: &section1step2Header,
			Type:   &section1step2Type,
		},
	}
	section1 := artifact.ClientSetupSection{
		Header: &header1,
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section1Steps,
	})

	header2 := "Push a version"
	section2step1Header := "Run this Helm push command in your terminal to push a chart in OCI form." +
		" Note: Make sure you add oci:// prefix to the repository URL."
	helmPushValue := "helm push <CHART_TGZ_FILE> oci://<HOSTNAME>/<REGISTRY_NAME>"
	section2step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &helmPushValue},
	}
	section2step1Type := artifact.ClientSetupStepTypeStatic
	section2Steps := []artifact.ClientSetupStep{
		{
			Header:   &section2step1Header,
			Commands: &section2step1Commands,
			Type:     &section2step1Type,
		},
	}
	section2 := artifact.ClientSetupSection{
		Header: &header2,
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section2Steps,
	})

	header3 := "Pull a version"
	section3step1Header := "Run this Helm command in your terminal to pull a specific chart version."
	var helmPullValue string
	if c.UntaggedImagesEnabled(ctx) {
		helmPullValue = "helm pull oci://<HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>@<DIGEST>"
	} else {
		helmPullValue = "helm pull oci://<HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME> --version <TAG>"
	}

	section3step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &helmPullValue},
	}
	section3step1Type := artifact.ClientSetupStepTypeStatic
	section3Steps := []artifact.ClientSetupStep{
		{
			Header:   &section3step1Header,
			Commands: &section3step1Commands,
			Type:     &section3step1Type,
		},
	}
	section3 := artifact.ClientSetupSection{
		Header: &header3,
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &section3Steps,
	})

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
		MainHeader: "Helm Client Setup",
		SecHeader:  "Follow these instructions to install/use Helm artifacts or compatible packages.",
		Sections:   sections,
	}
	return clientSetupDetails
}

// TODO: Remove StringPtr / see why it is used.
func (c *APIController) generateMavenClientSetupDetail(
	ctx context.Context,
	artifactName *artifact.ArtifactParam,
	version *artifact.VersionParam,
	registryRef string,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenStepType := artifact.ClientSetupStepTypeGenerateToken
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousMavenClientSetupDetails(staticStepType)
	} else {
		clientSetupDetails = c.getMavenClientSetupDetails(generateTokenStepType, staticStepType, registryType)
	}

	groupID := ""
	if artifactName != nil {
		parts := strings.Split(string(*artifactName), ":")
		if len(parts) == 2 {
			groupID = parts[0]
			*artifactName = artifact.ArtifactParam(parts[1])
		}
	}
	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "maven")

	//nolint:lll
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, artifactName, version, registryURL,
		groupID, "")

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousMavenClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
) artifact.ClientSetupDetails {
	mavenSection1 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("1. Pull a Maven Package"),
		SecHeader: utils.StringPtr("Set default repository in your pom.xml file."),
	}
	_ = mavenSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("To set default registry in your pom.xml file by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<repositories>\n  <repository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL></url>\n    <releases>\n      <enabled>true</enabled>\n      <updatePolicy>always</updatePolicy>\n    </releases>\n    <snapshots>\n      <enabled>true</enabled>\n      <updatePolicy>always</updatePolicy>\n    </snapshots>\n  </repository>\n</repositories>"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("Add a dependency to the project's pom.xml (replace <GROUP_ID>, <ARTIFACT_ID> & <VERSION> with your own):"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<dependency>\n  <groupId><GROUP_ID></groupId>\n  <artifactId><ARTIFACT_ID></artifactId>\n  <version><VERSION></version>\n</dependency>"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install dependencies in pom.xml file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("mvn install"),
					},
				},
			},
		},
	})

	gradleSection1 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("1. Pull a Gradle Package"),
		SecHeader: utils.StringPtr("Set default repository in your build.gradle file."),
	}
	_ = gradleSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				//nolint:lll
				Header: utils.StringPtr("Set the default registry in your project’s build.gradle by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("repositories{\n    maven{\n      url \"<REGISTRY_URL>\"\n\n   }\n}"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add a dependency to the project’s build.gradle"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dependencies {\n  implementation '<GROUP_ID>:<ARTIFACT_ID>:<VERSION>'\n}"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install dependencies in build.gradle file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("gradlew build     // Linux or OSX\n gradlew.bat build  // Windows"),
					},
				},
			},
		},
	})

	sbtSection1 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("1. Pull a Sbt/Scala Package"),
		SecHeader: utils.StringPtr("Set default repository in your build.sbt file."),
	}
	_ = sbtSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				//nolint:lll
				Header: utils.StringPtr("Set the default registry in your project’s build.sbt by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("resolver += \"Harness Registry\" at \"<REGISTRY_URL>\"\n"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add a dependency to the project’s build.sbt"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("libraryDependencies += \"<GROUP_ID>\" % \"<ARTIFACT_ID>\" % \"<VERSION>\""),
					},
				},
			},
			{
				Header: utils.StringPtr("Install dependencies in build.sbt file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sbt update"),
					},
				},
			},
		},
	})

	section2 := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: utils.StringPtr("Maven"),
				Sections: &[]artifact.ClientSetupSection{
					mavenSection1,
				},
			},
			{
				Header: utils.StringPtr("Gradle"),
				Sections: &[]artifact.ClientSetupSection{
					gradleSection1,
				},
			},
			{
				Header: utils.StringPtr("Sbt/Scala"),
				Sections: &[]artifact.ClientSetupSection{
					sbtSection1,
				},
			},
		},
	}

	_ = section2.FromTabSetupStepConfig(config)

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Maven Client Setup",
		SecHeader:  "Follow these instructions use Maven artifacts or compatible packages.",
		Sections: []artifact.ClientSetupSection{
			section2,
		},
	}
	return clientSetupDetails
}

func (c *APIController) getMavenClientSetupDetails(
	generateTokenStepType artifact.ClientSetupStepType,
	staticStepType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("1. Generate Identity Token"),
		//nolint:lll
		SecHeader: utils.StringPtr("An identity token will serve as the password for uploading and downloading artifacts."),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Generate an identity token"),
				Type:   &generateTokenStepType,
			},
		},
	})

	mavenSection1 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("2. Pull a Maven Package"),
		SecHeader: utils.StringPtr("Set default repository in your pom.xml file."),
	}
	_ = mavenSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("To set default registry in your pom.xml file by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<repositories>\n  <repository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL></url>\n    <releases>\n      <enabled>true</enabled>\n      <updatePolicy>always</updatePolicy>\n    </releases>\n    <snapshots>\n      <enabled>true</enabled>\n      <updatePolicy>always</updatePolicy>\n    </snapshots>\n  </repository>\n</repositories>"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("Copy the following your ~/ .m2/settings.xml file for MacOs, or $USERPROFILE$\\ .m2\\settings.xml for Windows to authenticate with token to pull from your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<settings>\n  <servers>\n    <server>\n      <id>maven-dev</id>\n      <username><USERNAME></username>\n      <password>identity-token</password>\n    </server>\n  </servers>\n</settings>"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("Add a dependency to the project's pom.xml (replace <GROUP_ID>, <ARTIFACT_ID> & <VERSION> with your own):"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<dependency>\n  <groupId><GROUP_ID></groupId>\n  <artifactId><ARTIFACT_ID></artifactId>\n  <version><VERSION></version>\n</dependency>"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install dependencies in pom.xml file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("mvn install"),
					},
				},
			},
		},
	})

	mavenSection2 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("3. Push a Maven Package"),
		SecHeader: utils.StringPtr("Set default repository in your pom.xml file."),
	}

	_ = mavenSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("To set default registry in your pom.xml file by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<distributionManagement>\n  <snapshotRepository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL></url>\n  </snapshotRepository>\n  <repository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL></url>\n  </repository>\n</distributionManagement>"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("Copy the following your ~/ .m2/setting.xml file for MacOs, or $USERPROFILE$\\ .m2\\settings.xml for Windows to authenticate with token to push to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<settings>\n  <servers>\n    <server>\n      <id>maven-dev</id>\n      <username><USERNAME></username>\n      <password>identity-token</password>\n    </server>\n  </servers>\n</settings>"),
					},
				},
			},
			{
				Header: utils.StringPtr("Publish package to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("mvn deploy"),
					},
				},
			},
		},
	})

	gradleSection1 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("2. Pull a Gradle Package"),
		SecHeader: utils.StringPtr("Set default repository in your build.gradle file."),
	}
	_ = gradleSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				//nolint:lll
				Header: utils.StringPtr("Set the default registry in your project’s build.gradle by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("repositories{\n    maven{\n      url \"<REGISTRY_URL>\"\n\n      credentials {\n         username \"<USERNAME>\"\n         password \"identity-token\"\n      }\n   }\n}"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("As this is a private registry, you’ll need to authenticate. Create or add to the ~/.gradle/gradle.properties file with the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("repositoryUser=<USERNAME>\nrepositoryPassword={{identity-token}}"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add a dependency to the project’s build.gradle"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dependencies {\n  implementation '<GROUP_ID>:<ARTIFACT_ID>:<VERSION>'\n}"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install dependencies in build.gradle file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("gradlew build     // Linux or OSX\n gradlew.bat build  // Windows"),
					},
				},
			},
		},
	})

	gradleSection2 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("3. Push a Gradle Package"),
		SecHeader: utils.StringPtr("Set default repository in your build.gradle file."),
	}

	_ = gradleSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add a maven publish plugin configuration to the project's build.gradle."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("publishing {\n    publications {\n        maven(MavenPublication) {\n            groupId = 'GROUP_ID'\n            artifactId = 'ARTIFACT_ID'\n            version = 'VERSION'\n\n            from components.java\n        }\n    }\n}"),
					},
				},
			},
			{
				Header: utils.StringPtr("Publish package to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("gradlew publish"),
					},
				},
			},
		},
	})

	sbtSection1 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("2. Pull a Sbt/Scala Package"),
		SecHeader: utils.StringPtr("Set default repository in your build.sbt file."),
	}
	_ = sbtSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Set the default registry in your project’s build.sbt by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("resolver += \"Harness Registry\" at \"<REGISTRY_URL>\"\ncredentials += Credentials(Path.userHome / \".sbt\" / \".Credentials\")"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("As this is a private registry, you’ll need to authenticate. Create or add to the ~/.sbt/.credentials file with the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("realm=Harness Registry\nhost=<LOGIN_HOSTNAME>\nuser=<USERNAME>\npassword={{identity-token}}"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add a dependency to the project’s build.sbt"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("libraryDependencies += \"<GROUP_ID>\" % \"<ARTIFACT_ID>\" % \"<VERSION>\""),
					},
				},
			},
			{
				Header: utils.StringPtr("Install dependencies in build.sbt file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sbt update"),
					},
				},
			},
		},
	})

	sbtSection2 := artifact.ClientSetupSection{
		Header:    utils.StringPtr("3. Push a Sbt/Scala Package"),
		SecHeader: utils.StringPtr("Set default repository in your build.sbt file."),
	}

	_ = sbtSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add publish configuration to the project’s build.sbt."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("publishTo := Some(\"Harness Registry\" at \"<REGISTRY_URL>\")"),
					},
				},
			},
			{
				Header: utils.StringPtr("Publish package to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sbt publish"),
					},
				},
			},
		},
	})

	section2 := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: utils.StringPtr("Maven"),
				Sections: &[]artifact.ClientSetupSection{
					mavenSection1,
				},
			},
			{
				Header: utils.StringPtr("Gradle"),
				Sections: &[]artifact.ClientSetupSection{
					gradleSection1,
				},
			},
			{
				Header: utils.StringPtr("Sbt/Scala"),
				Sections: &[]artifact.ClientSetupSection{
					sbtSection1,
				},
			},
		},
	}
	if registryType == artifact.RegistryTypeVIRTUAL {
		for i, remoteSection := range []artifact.ClientSetupSection{mavenSection2, gradleSection2, sbtSection2} {
			*(*config.Tabs)[i].Sections = append(*(*config.Tabs)[i].Sections, remoteSection)
		}
	}

	_ = section2.FromTabSetupStepConfig(config)

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Maven Client Setup",
		SecHeader:  "Follow these instructions to install/use Maven artifacts or compatible packages.",
		Sections: []artifact.ClientSetupSection{
			section1,
			section2,
		},
	}
	return clientSetupDetails
}

func (c *APIController) generateRpmClientSetupDetail(
	ctx context.Context,
	artifactName *artifact.ArtifactParam,
	version *artifact.VersionParam,
	registryRef string,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenStepType := artifact.ClientSetupStepTypeGenerateToken
	var clientSetupDetails artifact.ClientSetupDetails
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email

	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousRpmClientSetupDetails(staticStepType)
	} else {
		clientSetupDetails = c.getRpmClientSetupDetails(generateTokenStepType, staticStepType)
	}

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "rpm")

	//nolint:lll
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, artifactName, version, registryURL,
		"", "")

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getRpmClientSetupDetails(
	generateTokenStepType artifact.ClientSetupStepType,
	staticStepType artifact.ClientSetupStepType,
) artifact.ClientSetupDetails {
	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("1. Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenStepType,
			},
		},
	})

	yumSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("2. Install a RPM Package"),
	}
	_ = yumSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or edit the .repo file."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo vi /etc/yum.repos.d/harness-<REGISTRY_NAME>.repo"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("[harness-<REGISTRY_NAME>]\n" +
							"name=harness-<REGISTRY_NAME>\n" +
							"baseurl=<REGISTRY_URL>\n" +
							"enabled=1\n" +
							"gpgcheck=0\n" +
							"username=<USERNAME>\n" +
							"password=*see step 1*\n"),
					},
				},
			},
			{
				Header: utils.StringPtr("Clear the YUM cache."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo yum clean all"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install package."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo yum install <ARTIFACT_NAME>"),
					},
				},
			},
		},
	})

	yumSection2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("3. Upload RPM Package"),
	}

	_ = yumSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("To upload a RPM artifact run the following cURL with your package file:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("curl --location --request PUT '<REGISTRY_URL>/' \\\n--form 'file=@\"<FILE_PATH>\"' \\\n--header '<AUTH_HEADER_PREFIX> <API_KEY>'"),
					},
				},
			},
		},
	})

	dnfSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("2. Install a RPM Package"),
	}
	_ = dnfSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or edit the .repo file."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo vi /etc/yum.repos.d/harness-<REGISTRY_NAME>.repo"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("[harness-<REGISTRY_NAME>]\n" +
							"name=harness-<REGISTRY_NAME>\n" +
							"baseurl=<REGISTRY_URL>\n" +
							"enabled=1\n" +
							"gpgcheck=0\n" +
							"username=<USERNAME>\n" +
							"password=*see step 1*\n"),
					},
				},
			},
			{
				Header: utils.StringPtr("Clear the DNF cache."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo dnf clean all"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install package."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo dnf install <ARTIFACT_NAME>"),
					},
				},
			},
		},
	})

	dnfSection2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("3. Upload RPM Package"),
	}

	_ = dnfSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("To upload a RPM artifact run the following cURL with your package file:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("curl --location --request PUT '<REGISTRY_URL>/' \\\n--form 'file=@\"<FILE_PATH>\"' \\\n--header '<AUTH_HEADER_PREFIX> <API_KEY>'"),
					},
				},
			},
		},
	})

	section2 := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: utils.StringPtr("YUM"),
				Sections: &[]artifact.ClientSetupSection{
					yumSection1,
					yumSection2,
				},
			},
			{
				Header: utils.StringPtr("DNF"),
				Sections: &[]artifact.ClientSetupSection{
					dnfSection1,
					dnfSection2,
				},
			},
		},
	}

	_ = section2.FromTabSetupStepConfig(config)

	return artifact.ClientSetupDetails{
		MainHeader: "RPM Client Setup",
		SecHeader:  "Follow these instructions to install/upload RPM packages.",
		Sections: []artifact.ClientSetupSection{
			section1,
			section2,
		},
	}
}

func (c *APIController) getAnonymousRpmClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
) artifact.ClientSetupDetails {
	yumSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("1. Install a RPM Package"),
	}
	_ = yumSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or edit the .repo file."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo vi /etc/yum.repos.d/harness-<REGISTRY_NAME>.repo"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("[harness-<REGISTRY_NAME>]\n" +
							"name=harness-<REGISTRY_NAME>\n" +
							"baseurl=<REGISTRY_URL>\n" +
							"enabled=1\n" +
							"gpgcheck=0\n",
						),
					},
				},
			},
			{
				Header: utils.StringPtr("Clear the YUM cache."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo yum clean all"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install package."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo yum install <ARTIFACT_NAME>"),
					},
				},
			},
		},
	})

	dnfSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("1. Install a RPM Package"),
	}
	_ = dnfSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or edit the .repo file."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo vi /etc/yum.repos.d/harness-<REGISTRY_NAME>.repo"),
					},
				},
			},
			{
				Header: utils.StringPtr("Add the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("[harness-<REGISTRY_NAME>]\n" +
							"name=harness-<REGISTRY_NAME>\n" +
							"baseurl=<REGISTRY_URL>\n" +
							"enabled=1\n" +
							"gpgcheck=0\n",
						),
					},
				},
			},
			{
				Header: utils.StringPtr("Clear the DNF cache."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo dnf clean all"),
					},
				},
			},
			{
				Header: utils.StringPtr("Install package."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("sudo dnf install <ARTIFACT_NAME>"),
					},
				},
			},
		},
	})

	section2 := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: utils.StringPtr("YUM"),
				Sections: &[]artifact.ClientSetupSection{
					yumSection1,
				},
			},
			{
				Header: utils.StringPtr("DNF"),
				Sections: &[]artifact.ClientSetupSection{
					dnfSection1,
				},
			},
		},
	}

	_ = section2.FromTabSetupStepConfig(config)

	return artifact.ClientSetupDetails{
		MainHeader: "RPM Client Setup",
		SecHeader:  "Follow these instructions to install RPM packages.",
		Sections: []artifact.ClientSetupSection{
			section2,
		},
	}
}

func (c *APIController) generatePythonClientSetupDetail(
	ctx context.Context,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousPythonClientSetupDetails(staticStepType, registryType)
	} else {
		clientSetupDetails = c.getPythonClientSetupDetails(staticStepType, generateTokenType, registryType)
	}

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "python")
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, registryURL, "",
		string(artifact.PackageTypePYTHON))

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousPythonClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or update your ~/.pypirc file with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("[distutils]\n" +
							"index-servers = harness\n\n" +
							"[harness]\n" +
							"repository = <REGISTRY_URL>"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using pip:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("pip install --index-url <UPLOAD_URL>/simple --no-deps <ARTIFACT_NAME>==<VERSION>"),
					},
				},
			},
		},
	})

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
		MainHeader: "Python Client Setup",
		SecHeader:  "Follow these instructions to install/use Python packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getPythonClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	generateTokenType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or update your ~/.pypirc file with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("[distutils]\n" +
							"index-servers = harness\n\n" +
							"[harness]\n" +
							"repository = <REGISTRY_URL>\n" +
							"username = <USERNAME>\n" +
							"password = *see step 2*"),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	// Publish section
	section2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Publish Package"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Build and publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("python -m build"),
					},
					{
						Value: utils.StringPtr("python -m twine upload --repository harness /path/to/files/*"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using pip:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("pip install --index-url <UPLOAD_URL>/simple --no-deps <ARTIFACT_NAME>==<VERSION>"),
					},
				},
			},
		},
	})

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
		MainHeader: "Python Client Setup",
		SecHeader:  "Follow these instructions to install/use Python packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) generateNugetClientSetupDetail(
	ctx context.Context,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousNugetClientSetupDetails(staticStepType)
	} else {
		clientSetupDetails = c.getNugetClientSetupDetails(staticStepType, generateTokenType, registryType)
	}

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "nuget")

	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, registryURL, "",
		string(artifact.PackageTypeNUGET))

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousNugetClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
) artifact.ClientSetupDetails {
	nugetSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure registry"),
	}
	_ = nugetSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add the Harness Registry as a package source:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("nuget sources add -Name harness -Source " +
							"<REGISTRY_URL>/index.json \n\n"),
					},
					{
						Label: utils.StringPtr("Note: For Nuget V2 Client, use this url: <REGISTRY_URL>/"),
						Value: utils.StringPtr("<REGISTRY_URL>/"),
					},
				},
			},
		},
	})

	dotnetSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure registry"),
	}
	_ = dotnetSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add the Harness Registry as a package source:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dotnet nuget add source  " +
							"<REGISTRY_URL>/index.json " +
							"--name harness \n\n"),
					},
					{
						Label: utils.StringPtr("Note: For Nuget V2 Client, use this url: <REGISTRY_URL>/"),
						Value: utils.StringPtr("<REGISTRY_URL>/"),
					},
				},
			},
		},
	})

	visualStudioSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Nuget Package Source"),
	}
	_ = visualStudioSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add below config in Nuget.Config file to add Harness Registry as package source:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<configuration>\n <packageSources>\n     <clear />\n     <add key=\"harness\" value=\"<REGISTRY_URL>/index.json\" />\n </packageSources>\n</configuration>"),
					},
				},
			},
		},
	})

	nugetSection3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}

	_ = nugetSection3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using nuget:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("nuget install <ARTIFACT_NAME> -Version <VERSION> -Source harness"),
					},
				},
			},
		},
	})

	dotnetSection3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}

	_ = dotnetSection3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add a package using dotnet:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dotnet package add <ARTIFACT_NAME> --version <VERSION> --source harness"),
					},
				},
			},
		},
	})
	section := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: utils.StringPtr("Nuget"),
				Sections: &[]artifact.ClientSetupSection{
					nugetSection1,
					nugetSection3,
				},
			},
			{
				Header: utils.StringPtr("Dotnet"),
				Sections: &[]artifact.ClientSetupSection{
					dotnetSection1,
					dotnetSection3,
				},
			},
			{
				Header: utils.StringPtr("Visual Studio"),
				Sections: &[]artifact.ClientSetupSection{
					visualStudioSection1,
				},
			},
		},
	}

	_ = section.FromTabSetupStepConfig(config)

	sections := []artifact.ClientSetupSection{
		section,
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Nuget Client Setup",
		SecHeader:  "Follow these instructions to install/use Nuget packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getNugetClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	generateTokenType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	nugetSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = nugetSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add the Harness Registry as a package source:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("nuget sources add -Name harness -Source " +
							"<REGISTRY_URL>/index.json " +
							"-Username <USERNAME> " +
							"-Password <TOKEN>\n\n"),
					},
					{
						Value: utils.StringPtr("nuget setapikey <TOKEN> -Source harness\n\n"),
					},
					{
						Label: utils.StringPtr("Note: For Nuget V2 Client, use this url: <REGISTRY_URL>/"),
						Value: utils.StringPtr("<REGISTRY_URL>/"),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	dotnetSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = dotnetSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add the Harness Registry as a package source:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dotnet nuget add source  " +
							"<REGISTRY_URL>/index.json " +
							"--name harness --username <USERNAME> " +
							"--password <TOKEN> --store-password-in-clear-text\n\n"),
					},
					{
						Label: utils.StringPtr("Note: For Nuget V2 Client, use this url: <REGISTRY_URL>/"),
						Value: utils.StringPtr("<REGISTRY_URL>/"),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	visualStudioSection1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Nuget Package Source"),
	}
	_ = visualStudioSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add below config in Nuget.Config file to add Harness Registry as package source:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<configuration>\n <packageSources>\n     <clear />\n     <add key=\"harness\" value=\"<REGISTRY_URL>/index.json\" />\n </packageSources>\n <packageSourceCredentials>\n     <harness>\n         <add key=\"Username\" value=\"<USERNAME>\" />\n         <add key=\"ClearTextPassword\" value=\"<TOKEN>\" />\n     </harness>\n </packageSourceCredentials>\n</configuration>"),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	nugetSection2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Publish Package"),
	}

	_ = nugetSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("nuget push <PACKAGE_FILE> -Source harness\n\n"),
					},
					{
						Label: utils.StringPtr("Note: To publish your package with nested directory," +
							" refer below command"),
						Value: utils.StringPtr(""),
					},
					{
						Value: utils.StringPtr("nuget push <PACKAGE_FILE> -Source <REGISTRY_URL>/<SUB_DIRECTORY>" +
							" -ApiKey <TOKEN>"),
					},
				},
			},
		},
	})

	dotnetSection2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Publish Package"),
	}

	_ = dotnetSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dotnet nuget push <PACKAGE_FILE> --api-key <TOKEN>" +
							" --source harness\n\n"),
					},
					{
						Label: utils.StringPtr("Note: To publish your package with nested directory," +
							" refer below command"),
						Value: utils.StringPtr(""),
					},
					{
						Value: utils.StringPtr("dotnet nuget push <PACKAGE_FILE>" +
							" --source <REGISTRY_URL>/<SUB_DIRECTORY> --api-key <TOKEN>"),
					},
				},
			},
		},
	})

	nugetSection3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}

	_ = nugetSection3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using nuget:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("nuget install <ARTIFACT_NAME> -Version <VERSION> -Source harness"),
					},
				},
			},
		},
	})

	dotnetSection3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}

	_ = dotnetSection3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Add a package using dotnet:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("dotnet package add <ARTIFACT_NAME> --version <VERSION> --source harness"),
					},
				},
			},
		},
	})
	section := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: utils.StringPtr("Nuget"),
				Sections: &[]artifact.ClientSetupSection{
					nugetSection1,
					nugetSection3,
				},
			},
			{
				Header: utils.StringPtr("Dotnet"),
				Sections: &[]artifact.ClientSetupSection{
					dotnetSection1,
					dotnetSection3,
				},
			},
			{
				Header: utils.StringPtr("Visual Studio"),
				Sections: &[]artifact.ClientSetupSection{
					visualStudioSection1,
				},
			},
		},
	}

	if registryType == artifact.RegistryTypeVIRTUAL {
		for i, remoteSection := range []artifact.ClientSetupSection{nugetSection2, dotnetSection2} {
			*(*config.Tabs)[i].Sections = append(*(*config.Tabs)[i].Sections, remoteSection)
		}
	}

	_ = section.FromTabSetupStepConfig(config)

	sections := []artifact.ClientSetupSection{
		section,
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Nuget Client Setup",
		SecHeader:  "Follow these instructions to install/use Nuget packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) generateGoClientSetupDetail(
	ctx context.Context,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousGoClientSetupDetails(staticStepType, registryType)
	} else {
		clientSetupDetails = c.getGoClientSetupDetails(staticStepType, generateTokenType, registryType)
	}

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "go")
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, registryURL, "",
		string(artifact.PackageTypeGO))

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousGoClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure registry"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr(`To resolve a Go package from this registry using Go, 
				first set your default Harness Go registry by running the following command:`),
				Type: &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr(`export GOPROXY="<UPLOAD_URL>"`),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using go client"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("go get <ARTIFACT_NAME>@<VERSION>"),
					},
				},
			},
		},
	})

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
		MainHeader: "Go Client Setup",
		SecHeader:  "Follow these instructions to install/use go packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getGoClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	generateTokenType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr(`To resolve a Go package from this registry using Go, 
				first set your default Harness Go registry by running the following command:`),
				Type: &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr(`export GOPROXY="<UPLOAD_URL>"`),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	// Publish section
	section2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Publish Package"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr(`To deploy a Go package into a Harness registry, 
				you need to run the following Harness CLI command from your project’s root directory:`),
				Type: &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("hns ar push go <REGISTRY_NAME> <ARTIFACT_VERSION> --pkg-url <LOGIN_HOSTNAME>"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using go client"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("go get <ARTIFACT_NAME>@<VERSION>"),
					},
				},
			},
		},
	})

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
		MainHeader: "Go Client Setup",
		SecHeader:  "Follow these instructions to install/use go packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) generateNpmClientSetupDetail(
	ctx context.Context,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	var clientSetupDetails artifact.ClientSetupDetails
	if auth.IsAnonymousSession(session) {
		clientSetupDetails = c.getAnonymousNpmClientSetupDetails(staticStepType, registryType)
	} else {
		clientSetupDetails = c.getNpmClientSetupDetails(staticStepType, generateTokenType, registryType)
	}

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "npm")
	registryURL = strings.TrimPrefix(registryURL, "http:")
	registryURL = strings.TrimPrefix(registryURL, "https:")
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, registryURL, "",
		string(artifact.PackageTypeNPM))

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) getAnonymousNpmClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure registry"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or update your ~/.npmrc file with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("registry=https:<REGISTRY_URL>/"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using npm"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("npm install <ARTIFACT_NAME>@<VERSION>"),
					},
				},
			},
		},
	})

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
		MainHeader: "NPM Client Setup",
		SecHeader:  "Follow these instructions to install/use NPM packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) getNpmClientSetupDetails(
	staticStepType artifact.ClientSetupStepType,
	generateTokenType artifact.ClientSetupStepType,
	registryType artifact.RegistryType,
) artifact.ClientSetupDetails {
	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Create or update your ~/.npmrc file with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("registry=https:<REGISTRY_URL>/"),
					},
					{
						Value: utils.StringPtr("<REGISTRY_URL>/:_authToken=<TOKEN>"),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	// Publish section
	section2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Publish Package"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Build and publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("npm run build\n"),
					},
					{
						Value: utils.StringPtr("npm publish"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Install a package using npm"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("npm install <ARTIFACT_NAME>@<VERSION>"),
					},
				},
			},
		},
	})

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
		MainHeader: "NPM Client Setup",
		SecHeader:  "Follow these instructions to install/use NPM packages from this registry.",
		Sections:   sections,
	}
	return clientSetupDetails
}

func (c *APIController) replacePlaceholders(
	ctx context.Context,
	clientSetupSections *[]artifact.ClientSetupSection,
	username string,
	regRef string,
	image *artifact.ArtifactParam,
	version *artifact.VersionParam,
	registryURL string,
	groupID string,
	pkgType string,
) {
	uploadURL := ""
	if pkgType == string(artifact.PackageTypePYTHON) || pkgType == string(artifact.PackageTypeGO) {
		session, _ := request.AuthSessionFrom(ctx)
		regURL, _ := url.Parse(registryURL)
		if auth.IsAnonymousSession(session) {
			uploadURL = fmt.Sprintf("%s://%s%s", regURL.Scheme, regURL.Host, regURL.Path)
		} else {
			// append username:password to the host
			regURL.User = url.UserPassword(username, "<TOKEN>")
			uploadURL = fmt.Sprintf("%s://%s:%s@%s%s", regURL.Scheme, regURL.User.Username(), "<TOKEN>", regURL.Host,
				regURL.Path)
		}
	}

	for i := range *clientSetupSections {
		tab, err := (*clientSetupSections)[i].AsTabSetupStepConfig()
		if err != nil || tab.Tabs == nil {
			//nolint:lll
			c.replacePlaceholdersInSection(ctx, &(*clientSetupSections)[i], username, regRef, image, version, pkgType,
				registryURL, groupID, uploadURL)
		} else {
			for j := range *tab.Tabs {
				c.replacePlaceholders(ctx, (*tab.Tabs)[j].Sections, username, regRef, image, version, registryURL, groupID,
					pkgType)
			}
			_ = (*clientSetupSections)[i].FromTabSetupStepConfig(tab)
		}
	}
}

func (c *APIController) replacePlaceholdersInSection(
	ctx context.Context,
	clientSetupSection *artifact.ClientSetupSection,
	username string,
	regRef string,
	image *artifact.ArtifactParam,
	version *artifact.VersionParam,
	pkgType string,
	registryURL string,
	groupID string,
	uploadURL string,
) {
	rootSpace, _, _ := paths.DisectRoot(regRef)
	_, registryName, _ := paths.DisectLeaf(regRef)
	var hostname string
	if pkgType == string(artifact.PackageTypeGENERIC) {
		hostname = c.URLProvider.PackageURL(ctx, regRef, "")
	} else {
		hostname = common.TrimURLScheme(c.URLProvider.RegistryURL(ctx, rootSpace))
	}

	sec, err := clientSetupSection.AsClientSetupStepConfig()
	if err != nil || sec.Steps == nil {
		return
	}
	for _, st := range *sec.Steps {
		if st.Commands == nil {
			continue
		}
		for j := range *st.Commands {
			c.replaceText(ctx, username, st, j, hostname, registryName, image, version, registryURL, groupID, uploadURL)
		}
	}
	_ = clientSetupSection.FromClientSetupStepConfig(sec)
}

func (c *APIController) replaceText(
	ctx context.Context,
	username string,
	st artifact.ClientSetupStep,
	i int,
	hostname string,
	repoName string,
	image *artifact.ArtifactParam,
	version *artifact.VersionParam,
	registryURL string,
	groupID string,
	uploadURL string,
) {
	if c.SetupDetailsAuthHeaderPrefix != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value,
			"<AUTH_HEADER_PREFIX>", c.SetupDetailsAuthHeaderPrefix))
	}
	if username != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<USERNAME>", username))
		if (*st.Commands)[i].Label != nil {
			(*st.Commands)[i].Label = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Label, "<USERNAME>",
				username))
		}
	}
	if groupID != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<GROUP_ID>", groupID))
	}
	if registryURL != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<REGISTRY_URL>",
			registryURL))
		if (*st.Commands)[i].Label != nil {
			(*st.Commands)[i].Label = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Label,
				"<REGISTRY_URL>", registryURL))
		}
	}
	if uploadURL != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<UPLOAD_URL>",
			uploadURL))
	}
	if hostname != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<HOSTNAME>", hostname))
	}
	if hostname != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value,
			"<LOGIN_HOSTNAME>", common.GetHost(ctx, hostname)))
	}
	if repoName != "" {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<REGISTRY_NAME>",
			repoName))
	}
	if image != nil {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<IMAGE_NAME>",
			string(*image)))
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<ARTIFACT_ID>",
			string(*image)))
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<ARTIFACT_NAME>",
			string(*image)))
	}
	if version != nil {
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<TAG>", string(*version)))
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<VERSION>", string(*version)))
		(*st.Commands)[i].Value = utils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<DIGEST>", string(*version)))
	}
}

func (c *APIController) generateHuggingFaceClientSetupDetail(
	ctx context.Context,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	// Configuration section
	section1 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Hugging Face Client"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Set up environment variables to connect to Artifact Registry:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("export HF_HUB_ETAG_TIMEOUT=86400\nexport HF_HUB_DOWNLOAD_TIMEOUT=86400\nexport HF_ENDPOINT=<REGISTRY_URL>"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("For Hugging Face client version 0.19.0 and above, this timeout setting enables model resolution via pipelines and tokenizer libraries."),
				Type:   &staticStepType,
			},
		},
	})

	// Authentication section
	section2 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Configure Authentication"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Set up authentication for the Hugging Face client:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: utils.StringPtr("export HF_TOKEN=<TOKEN>"),
					},
				},
			},
			{
				Header: utils.StringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	// Deploy Model section
	section3 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Deploy Model"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Upload a model to Artifact Registry using the huggingface_hub library:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("from huggingface_hub import HfApi\napi = HfApi()\napi.upload_folder(\n    folder_path=\"<folder_name>\", # local folder containing model files\n    repo_id=\"<ARTIFACT_NAME>\", # name for the model in the registry\n    revision=\"<VERSION>\", # model version (defaults to 'main')\n    repo_type=\"model\"\n)"),
					},
				},
			},
		},
	})

	// Deploy Dataset section
	section4 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Deploy Dataset"),
	}
	_ = section4.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Upload a dataset to Artifact Registry using the huggingface_hub library:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("from huggingface_hub import HfApi\napi = HfApi()\napi.upload_folder(\n    folder_path=\"<folder_name>\", # local folder containing dataset files\n    repo_id=\"<ARTIFACT_NAME>\", # name for the dataset in the registry\n    revision=\"<VERSION>\", # dataset version (defaults to 'main')\n    repo_type=\"dataset\"\n)"),
					},
				},
			},
		},
	})

	// Resolve Model section
	section5 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Resolve Model"),
	}
	_ = section5.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Download a model from Artifact Registry:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("from huggingface_hub import snapshot_download\nsnapshot_download(\n    repo_id=\"<ARTIFACT_NAME>\", revision=\"<VERSION>\", etag_timeout=86400\n)"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("With Hugging Face client version 0.19.0+ and HF_HUB_ETAG_TIMEOUT set, you can resolve models with transformers, diffusers, and other libraries."),
				Type:   &staticStepType,
			},
		},
	})

	// Resolve Dataset section
	section6 := artifact.ClientSetupSection{
		Header: utils.StringPtr("Resolve Dataset"),
	}
	_ = section6.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: utils.StringPtr("Artifact Registry supports resolving datasets" +
					" using Hugging Face dataset libraries:"),
				Type: &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("from datasets import load_dataset\ndataset = load_dataset(\"<ARTIFACT_NAME>\")"),
					},
				},
			},
			{
				Header: utils.StringPtr("Or download an entire dataset repository using the snapshot_download API:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: utils.StringPtr("from huggingface_hub import snapshot_download\nsnapshot_download(\n    repo_id=\"<ARTIFACT_NAME>\", revision=\"<VERSION>\", repo_type=\"dataset\", etag_timeout=86400\n)"),
					},
				},
			},
			{
				//nolint:lll
				Header: utils.StringPtr("Note: Artifact Registry fully caches only datasets hosted directly on Hugging Face, not those referencing external sources."),
				Type:   &staticStepType,
			},
		},
	})

	sections := []artifact.ClientSetupSection{section1}
	if !auth.IsAnonymousSession(session) {
		sections = append(sections, section2, section3, section4)
	}
	sections = append(sections, section5, section6)

	if registryType == artifact.RegistryTypeUPSTREAM {
		// For upstream registry, only include configuration, authentication, and resolve sections
		sections = []artifact.ClientSetupSection{section1}
		if !auth.IsAnonymousSession(session) {
			sections = append(sections, section2)
		}
		sections = append(sections, section5, section6)
	}

	clientSetupDetails := artifact.ClientSetupDetails{
		MainHeader: "Hugging Face Client Setup",
		SecHeader:  "Follow these instructions to interact with Hugging Face models and datasets in this registry.",
		Sections:   sections,
	}

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "huggingface")
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, registryURL, "",
		string(artifact.PackageTypeHUGGINGFACE))

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}
