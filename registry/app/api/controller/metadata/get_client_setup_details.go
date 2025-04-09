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
	"net/http"
	"net/url"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/common"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetClientSetupDetails(
	ctx context.Context,
	r artifact.GetClientSetupDetailsRequestObject,
) (artifact.GetClientSetupDetailsResponseObject, error) {
	regRefParam := r.RegistryRef
	imageParam := r.Params.Artifact
	tagParam := r.Params.Version

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

	reg, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if err != nil {
		return artifact.GetClientSetupDetails404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this ref"),
			),
		}, err
	}

	packageType := string(reg.PackageType)

	response := c.GenerateClientSetupDetails(
		ctx, packageType, imageParam, tagParam, regInfo.RegistryRef,
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
	tag *artifact.VersionParam,
	registryRef string,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	session, _ := request.AuthSessionFrom(ctx)
	username := session.Principal.Email
	loginUsernameLabel := "Username: <USERNAME>"
	loginUsernameValue := "<USERNAME>"
	loginPasswordLabel := "Password: *see step 2*"
	blankString := ""
	switch packageType {
	case string(artifact.PackageTypeMAVEN):
		return c.generateMavenClientSetupDetail(ctx, image, tag, registryRef, username, registryType)
	case string(artifact.PackageTypeHELM):
		return c.generateHelmClientSetupDetail(ctx, blankString, loginUsernameLabel, loginUsernameValue,
			loginPasswordLabel, username, registryRef, image, tag, registryType)
	case string(artifact.PackageTypeGENERIC):
		return c.generateGenericClientSetupDetail(ctx, blankString, registryRef, image, tag, registryType)
	case string(artifact.PackageTypePYTHON):
		return c.generatePythonClientSetupDetail(ctx, registryRef, username, image, tag, registryType)
	case string(artifact.PackageTypeNPM):
		return c.generateNpmClientSetupDetail(ctx, registryRef, username, image, tag, registryType)
	case string(artifact.PackageTypeDOCKER):
		return c.generateDockerClientSetupDetail(ctx, blankString, loginUsernameLabel, loginUsernameValue,
			loginPasswordLabel, registryType,
			username, registryRef, image, tag)
	default:
		log.Debug().Ctx(ctx).Msgf("Unknown package type for client details: %s", packageType)
		return nil
	}
}

func (c *APIController) generateDockerClientSetupDetail(
	ctx context.Context,
	blankString string,
	loginUsernameLabel string,
	loginUsernameValue string,
	loginPasswordLabel string,
	registryType artifact.RegistryType,
	username string,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
) *artifact.ClientSetupDetailsResponseJSONResponse {
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
	dockerPullValue := "docker pull <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<TAG>"
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
	dockerTagValue := "docker tag <IMAGE_NAME>:<TAG> <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<TAG>"
	section3step1Commands := []artifact.ClientSetupStepCommand{
		{Label: &blankString, Value: &dockerTagValue},
	}
	section3step1Type := artifact.ClientSetupStepTypeStatic
	section3step2Header := "Run this Docker command in your terminal to push the image."
	dockerPushValue := "docker push <HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME>:<TAG>"
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

	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, "", "", "")

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
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
	section2step1Header := "Run this curl command in your terminal to push the artifact."
	//nolint:lll
	pushValue := "curl --location --request PUT '<HOSTNAME>/<REGISTRY_NAME>/<ARTIFACT_NAME>/<VERSION>' \\\n--form 'filename=\"<FILENAME>\"' \\\n--form 'file=@\"<FILE_PATH>\"' \\\n--form 'description=\"<DESC>\"' \\\n--header 'x-api-key: <API_KEY>'"
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
	section3step1Header := "Run this command in your terminal to download the artifact."
	//nolint:lll
	pullValue := "curl --location '<HOSTNAME>/<REGISTRY_NAME>/<ARTIFACT_NAME>:<VERSION>:<FILENAME>' --header 'x-api-key: <API_KEY>' " +
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
		MainHeader: "Generic Client Setup",
		SecHeader:  "Follow these instructions to install/use Generic artifacts or compatible packages.",
		Sections:   sections,
	}
	//nolint:lll
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, "", registryRef, image, tag, "", "",
		string(artifact.PackageTypeGENERIC))
	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

//nolint:lll
func (c *APIController) generateHelmClientSetupDetail(
	ctx context.Context,
	blankString string,
	loginUsernameLabel string,
	loginUsernameValue string,
	loginPasswordLabel string,
	username string,
	registryRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
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
	helmPullValue := "helm pull oci://<HOSTNAME>/<REGISTRY_NAME>/<IMAGE_NAME> --version <TAG>"
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

	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, "", "", "")

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) generateMavenClientSetupDetail(
	ctx context.Context,
	artifactName *artifact.ArtifactParam,
	version *artifact.VersionParam,
	registryRef string,
	username string,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenStepType := artifact.ClientSetupStepTypeGenerateToken

	section1 := artifact.ClientSetupSection{
		Header:    stringPtr("1. Generate Identity Token"),
		SecHeader: stringPtr("An identity token will serve as the password for uploading and downloading artifacts."),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Generate an identity token"),
				Type:   &generateTokenStepType,
			},
		},
	})

	mavenSection1 := artifact.ClientSetupSection{
		Header:    stringPtr("2. Pull a Maven Package"),
		SecHeader: stringPtr("Set default repository in your pom.xml file."),
	}
	_ = mavenSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("To set default registry in your pom.xml file by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("<repositories>\n  <repository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL>/<REGISTRY_NAME></url>\n    <releases>\n      <enabled>true</enabled>\n      <updatePolicy>always</updatePolicy>\n    </releases>\n    <snapshots>\n      <enabled>true</enabled>\n      <updatePolicy>always</updatePolicy>\n    </snapshots>\n  </repository>\n</repositories>"),
					},
				},
			},
			{
				//nolint:lll
				Header: stringPtr("Copy the following your ~/ .m2/settings.xml file for MacOs, or $USERPROFILE$\\ .m2\\settings.xml for Windows to authenticate with token to pull from your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("<settings>\n  <servers>\n    <server>\n      <id>maven-dev</id>\n      <username><USERNAME></username>\n      <password>identity-token</password>\n    </server>\n  </servers>\n</settings>"),
					},
				},
			},
			{
				//nolint:lll
				Header: stringPtr("Add a dependency to the project's pom.xml (replace <GROUP_ID>, <ARTIFACT_ID> & <VERSION> with your own):"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("<dependency>\n  <groupId><GROUP_ID></groupId>\n  <artifactId><ARTIFACT_ID></artifactId>\n  <version><VERSION></version>\n</dependency>"),
					},
				},
			},
			{
				Header: stringPtr("Install dependencies in pom.xml file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("mvn install"),
					},
				},
			},
		},
	})

	mavenSection2 := artifact.ClientSetupSection{
		Header:    stringPtr("3. Push a Maven Package"),
		SecHeader: stringPtr("Set default repository in your pom.xml file."),
	}

	_ = mavenSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("To set default registry in your pom.xml file by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("<distributionManagement>\n  <snapshotRepository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL>/<REGISTRY_NAME></url>\n  </snapshotRepository>\n  <repository>\n    <id>maven-dev</id>\n    <url><REGISTRY_URL>/<REGISTRY_NAME></url>\n  </repository>\n</distributionManagement>"),
					},
				},
			},
			{
				//nolint:lll
				Header: stringPtr("Copy the following your ~/ .m2/setting.xml file for MacOs, or $USERPROFILE$\\ .m2\\settings.xml for Windows to authenticate with token to push to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("<settings>\n  <servers>\n    <server>\n      <id>maven-dev</id>\n      <username><USERNAME></username>\n      <password>identity-token</password>\n    </server>\n  </servers>\n</settings>"),
					},
				},
			},
			{
				Header: stringPtr("Publish package to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("mvn deploy"),
					},
				},
			},
		},
	})

	gradleSection1 := artifact.ClientSetupSection{
		Header:    stringPtr("2. Pull a Gradle Package"),
		SecHeader: stringPtr("Set default repository in your build.gradle file."),
	}
	_ = gradleSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Set the default registry in your project’s build.gradle by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("repositories{\n    maven{\n      url \"<REGISTRY_URL>/<REGISTRY_NAME>\"\n\n      credentials {\n         username \"<USERNAME>\"\n         password \"identity-token\"\n      }\n   }\n}"),
					},
				},
			},
			{
				//nolint:lll
				Header: stringPtr("As this is a private registry, you’ll need to authenticate. Create or add to the ~/.gradle/gradle.properties file with the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("repositoryUser=<USERNAME>\nrepositoryPassword={{identity-token}}"),
					},
				},
			},
			{
				Header: stringPtr("Add a dependency to the project’s build.gradle"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("dependencies {\n  implementation '<GROUP_ID>:<ARTIFACT_ID>:<VERSION>'\n}"),
					},
				},
			},
			{
				Header: stringPtr("Install dependencies in build.gradle file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("gradlew build     // Linux or OSX\n gradlew.bat build  // Windows"),
					},
				},
			},
		},
	})

	gradleSection2 := artifact.ClientSetupSection{
		Header:    stringPtr("3. Push a Gradle Package"),
		SecHeader: stringPtr("Set default repository in your build.gradle file."),
	}

	_ = gradleSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Add a maven publish plugin configuration to the project's build.gradle."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("publishing {\n    publications {\n        maven(MavenPublication) {\n            groupId = 'GROUP_ID'\n            artifactId = 'ARTIFACT_ID'\n            version = 'VERSION'\n\n            from components.java\n        }\n    }\n}"),
					},
				},
			},
			{
				Header: stringPtr("Publish package to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("gradlew publish"),
					},
				},
			},
		},
	})

	sbtSection1 := artifact.ClientSetupSection{
		Header:    stringPtr("2. Pull a Sbt/Scala Package"),
		SecHeader: stringPtr("Set default repository in your build.sbt file."),
	}
	_ = sbtSection1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Set the default registry in your project’s build.sbt by adding the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("resolver += \"Harness Registry\" at \"<REGISTRY_URL>/<REGISTRY_NAME>\"\ncredentials += Credentials(Path.userHome / \".sbt\" / \".Credentials\")"),
					},
				},
			},
			{
				//nolint:lll
				Header: stringPtr("As this is a private registry, you’ll need to authenticate. Create or add to the ~/.sbt/.credentials file with the following:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						//nolint:lll
						Value: stringPtr("realm=Harness Registry\nhost=<LOGIN_HOSTNAME>\nuser=<USERNAME>\npassword={{identity-token}}"),
					},
				},
			},
			{
				Header: stringPtr("Add a dependency to the project’s build.sbt"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("libraryDependencies += \"<GROUP_ID>\" % \"<ARTIFACT_ID>\" % \"<VERSION>\""),
					},
				},
			},
			{
				Header: stringPtr("Install dependencies in build.sbt file"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("sbt update"),
					},
				},
			},
		},
	})

	sbtSection2 := artifact.ClientSetupSection{
		Header:    stringPtr("3. Push a Sbt/Scala Package"),
		SecHeader: stringPtr("Set default repository in your build.sbt file."),
	}

	_ = sbtSection2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Add publish configuration to the project’s build.sbt."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("publishTo := Some(\"Harness Registry\" at \"<REGISTRY_URL>/<REGISTRY_NAME>\")"),
					},
				},
			},
			{
				Header: stringPtr("Publish package to your Maven registry."),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("sbt publish"),
					},
				},
			},
		},
	})

	section2 := artifact.ClientSetupSection{}
	config := artifact.TabSetupStepConfig{
		Tabs: &[]artifact.TabSetupStep{
			{
				Header: stringPtr("Maven"),
				Sections: &[]artifact.ClientSetupSection{
					mavenSection1,
				},
			},
			{
				Header: stringPtr("Gradle"),
				Sections: &[]artifact.ClientSetupSection{
					gradleSection1,
				},
			},
			{
				Header: stringPtr("Sbt/Scala"),
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
	groupID := ""
	if artifactName != nil {
		parts := strings.Split(string(*artifactName), ":")
		if len(parts) == 2 {
			groupID = parts[0]
			*artifactName = artifact.ArtifactParam(parts[1])
		}
	}

	rootSpace, _, _ := paths.DisectRoot(registryRef)
	registryURL := c.URLProvider.RegistryURL(ctx, "maven", rootSpace)

	//nolint:lll
	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, artifactName, version, registryURL,
		groupID, "")

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) generatePythonClientSetupDetail(
	ctx context.Context,
	registryRef string,
	username string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken

	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: stringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Create or update your ~/.pypirc file with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("[distutils]\n" +
							"index-servers = harness\n\n" +
							"[harness]\n" +
							"repository = <REGISTRY_URL>\n" +
							"username = <USERNAME>\n" +
							"password = *see step 2*"),
					},
				},
			},
			{
				Header: stringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	// Publish section
	section2 := artifact.ClientSetupSection{
		Header: stringPtr("Publish Package"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Build and publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("python -m build"),
					},
					{
						Value: stringPtr("python -m twine upload --repository harness /path/to/files/*"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: stringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Install a package using pip:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("pip install --index-url <UPLOAD_URL>/simple --no-deps <ARTIFACT_NAME>==<VERSION>"),
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

	registryURL := c.URLProvider.PackageURL(ctx, registryRef, "python")

	c.replacePlaceholders(ctx, &clientSetupDetails.Sections, username, registryRef, image, tag, registryURL, "",
		string(artifact.PackageTypePYTHON))

	return &artifact.ClientSetupDetailsResponseJSONResponse{
		Data:   clientSetupDetails,
		Status: artifact.StatusSUCCESS,
	}
}

func (c *APIController) generateNpmClientSetupDetail(
	ctx context.Context,
	registryRef string,
	username string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryType artifact.RegistryType,
) *artifact.ClientSetupDetailsResponseJSONResponse {
	staticStepType := artifact.ClientSetupStepTypeStatic
	generateTokenType := artifact.ClientSetupStepTypeGenerateToken

	// Authentication section
	section1 := artifact.ClientSetupSection{
		Header: stringPtr("Configure Authentication"),
	}
	_ = section1.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Create or update your ~/.npmrc file with the following content:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("npm set registry https:<REGISTRY_URL>/\n\n" +
							"npm set <REGISTRY_URL>/:_authToken <TOKEN>"),
					},
				},
			},
			{
				Header: stringPtr("Generate an identity token for authentication"),
				Type:   &generateTokenType,
			},
		},
	})

	// Publish section
	section2 := artifact.ClientSetupSection{
		Header: stringPtr("Publish Package"),
	}
	_ = section2.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Build and publish your package:"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("npm run build\n"),
					},
					{
						Value: stringPtr("npm publish"),
					},
				},
			},
		},
	})

	// Install section
	section3 := artifact.ClientSetupSection{
		Header: stringPtr("Install Package"),
	}
	_ = section3.FromClientSetupStepConfig(artifact.ClientSetupStepConfig{
		Steps: &[]artifact.ClientSetupStep{
			{
				Header: stringPtr("Install a package using npm"),
				Type:   &staticStepType,
				Commands: &[]artifact.ClientSetupStepCommand{
					{
						Value: stringPtr("npm install <ARTIFACT_NAME>@<VERSION>"),
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

func (c *APIController) replacePlaceholders(
	ctx context.Context,
	clientSetupSections *[]artifact.ClientSetupSection,
	username string,
	regRef string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryURL string,
	groupID string,
	pkgType string,
) {
	uploadURL := ""
	if pkgType == string(artifact.PackageTypePYTHON) {
		regURL, _ := url.Parse(registryURL)
		// append username:password to the host
		regURL.User = url.UserPassword(username, "identity-token")
		uploadURL = regURL.String()
	}

	for i := range *clientSetupSections {
		tab, err := (*clientSetupSections)[i].AsTabSetupStepConfig()
		if err != nil || tab.Tabs == nil {
			//nolint:lll
			c.replacePlaceholdersInSection(ctx, &(*clientSetupSections)[i], username, regRef, image, tag, pkgType,
				registryURL, groupID, uploadURL)
		} else {
			for j := range *tab.Tabs {
				c.replacePlaceholders(ctx, (*tab.Tabs)[j].Sections, username, regRef, image, tag, registryURL, groupID,
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
	tag *artifact.VersionParam,
	pkgType string,
	registryURL string,
	groupID string,
	uploadURL string,
) {
	rootSpace, _, _ := paths.DisectRoot(regRef)
	_, registryName, _ := paths.DisectLeaf(regRef)
	var hostname string
	if pkgType == string(artifact.PackageTypeGENERIC) {
		hostname = c.URLProvider.RegistryURL(ctx, rootSpace, "generic")
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
			replaceText(username, st, j, hostname, registryName, image, tag, registryURL, groupID, uploadURL)
		}
	}
	_ = clientSetupSection.FromClientSetupStepConfig(sec)
}

func replaceText(
	username string,
	st artifact.ClientSetupStep,
	i int,
	hostname string,
	repoName string,
	image *artifact.ArtifactParam,
	tag *artifact.VersionParam,
	registryURL string,
	groupID string,
	uploadURL string,
) {
	if username != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<USERNAME>", username))
		if (*st.Commands)[i].Label != nil {
			(*st.Commands)[i].Label = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Label, "<USERNAME>", username))
		}
	}
	if groupID != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<GROUP_ID>", groupID))
	}
	if registryURL != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<REGISTRY_URL>", registryURL))
	}
	if uploadURL != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<UPLOAD_URL>", uploadURL))
	}
	if hostname != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<HOSTNAME>", hostname))
	}
	if hostname != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value,
			"<LOGIN_HOSTNAME>", common.GetHost(hostname)))
	}
	if repoName != "" {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<REGISTRY_NAME>", repoName))
	}
	if image != nil {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<IMAGE_NAME>",
			string(*image)))
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<ARTIFACT_ID>",
			string(*image)))
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<ARTIFACT_NAME>",
			string(*image)))
	}
	if tag != nil {
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<TAG>", string(*tag)))
		(*st.Commands)[i].Value = stringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<VERSION>", string(*tag)))
	}
}

func stringPtr(s string) *string {
	return &s
}
