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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/harness/gitness/types"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// convertOptionsToEnvVariables converts the option keys to standardised env variables using the
// devcontainer specification to ensure uniformity in the naming and casing of the env variables.
// eg: 217yat%_fg -> _YAT_FG
// Reference: https://containers.dev/implementors/features/#option-resolution
func convertOptionsToEnvVariables(str string) string {
	// Replace all non-alphanumeric characters (excluding underscores) with '_'
	reNonAlnum := regexp.MustCompile(`[^\w_]`)
	str = reNonAlnum.ReplaceAllString(str, "_")

	// Replace leading digits or underscores with a single '_'
	reLeadingDigitsOrUnderscores := regexp.MustCompile(`^[\d_]+`)
	str = reLeadingDigitsOrUnderscores.ReplaceAllString(str, "_")

	// Convert the string to uppercase
	str = strings.ToUpper(str)

	return str
}

// BuildWithFeatures builds a docker image using the provided image as the base image.
// It sets some common env variables using the ARG instruction.
// For every feature, it copies the containerEnv variables using the ENV instruction and passes the resolved options
// as env variables in the RUN instruction. It further executes the install script.
func BuildWithFeatures(
	ctx context.Context,
	dockerClient *client.Client,
	imageName string,
	features []*types.ResolvedFeature,
	gitspaceInstanceIdentifier string,
	containerUser string,
	remoteUser string,
	containerUserHomeDir string,
	remoteUserHomeDir string,
) (string, string, error) {
	buildContextPath := getGitspaceInstanceDirectory(gitspaceInstanceIdentifier)

	defer func() {
		err := os.RemoveAll(buildContextPath)
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to remove build context directory %s", buildContextPath)
		}
	}()

	dockerFileContent, err := generateDockerFileWithFeatures(imageName, features, buildContextPath, containerUser,
		containerUserHomeDir, remoteUser, remoteUserHomeDir)
	if err != nil {
		return "", "", err
	}

	buildContext, err := packBuildContextDirectory(buildContextPath)
	if err != nil {
		return "", dockerFileContent, err
	}

	newImageName := "gitspace-with-features:" + gitspaceInstanceIdentifier
	buildRes, imageBuildErr := dockerClient.ImageBuild(ctx, buildContext, dockerTypes.ImageBuildOptions{
		SuppressOutput: false,
		Tags:           []string{newImageName},
		Version:        dockerTypes.BuilderBuildKit,
	})

	defer func() {
		if buildRes.Body != nil {
			closeErr := buildRes.Body.Close()
			if closeErr != nil {
				log.Ctx(ctx).Err(closeErr).Msg("failed to close docker image build response body")
			}
		}
	}()

	if imageBuildErr != nil {
		return "", dockerFileContent, imageBuildErr
	}

	_, err = io.Copy(io.Discard, buildRes.Body)
	if err != nil {
		return "", dockerFileContent, err
	}

	imagePresentLocally, err := IsImagePresentLocally(ctx, newImageName, dockerClient)
	if err != nil {
		return "", dockerFileContent, err
	}
	if !imagePresentLocally {
		return "", dockerFileContent, fmt.Errorf("error during docker build, image %s not present", newImageName)
	}

	return newImageName, dockerFileContent, nil
}

func packBuildContextDirectory(path string) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(path, file)
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.Mode().IsRegular() {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(tw, f)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk path %q: %w", path, err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	return &buf, nil
}

// generateDockerFileWithFeatures creates and saves a dockerfile inside the build context directory.
func generateDockerFileWithFeatures(
	imageName string,
	features []*types.ResolvedFeature,
	buildContextPath string,
	containerUser string,
	remoteUser string,
	containerUserHomeDir string,
	remoteUserHomeDir string,
) (string, error) {
	dockerFile := fmt.Sprintf(`FROM %s
ARG %s=%s
ARG %s=%s
ARG %s=%s
ARG %s=%s
COPY ./devcontainer-features %s`,
		imageName, convertOptionsToEnvVariables("_CONTAINER_USER"), containerUser,
		convertOptionsToEnvVariables("_REMOTE_USER"), remoteUser,
		convertOptionsToEnvVariables("_CONTAINER_USER_HOME"), containerUserHomeDir,
		convertOptionsToEnvVariables("_REMOTE_USER_HOME"), remoteUserHomeDir,
		"/tmp/devcontainer-features")

	for _, feature := range features {
		if len(feature.DownloadedFeature.DevcontainerFeatureConfig.ContainerEnv) > 0 {
			envVariables := ""
			for key, value := range feature.DownloadedFeature.DevcontainerFeatureConfig.ContainerEnv {
				envVariables += " " + key + "=" + value
			}
			dockerFile += fmt.Sprintf("\nENV%s", envVariables)
		}

		finalOptionsMap := make(map[string]string)
		for key, value := range feature.ResolvedOptions {
			finalOptionsMap[convertOptionsToEnvVariables(key)] = value
		}

		optionEnvVariables := ""
		for key, value := range finalOptionsMap {
			optionEnvVariables += " " + key + "=" + value
		}

		installScriptPath := filepath.Join("/tmp/devcontainer-features",
			getFeatureFolderNameWithTag(feature.DownloadedFeature.FeatureFolderName, feature.DownloadedFeature.Tag),
			feature.DownloadedFeature.FeatureFolderName, "install.sh")
		dockerFile += fmt.Sprintf("\nRUN%s chmod +x %s && %s",
			optionEnvVariables, installScriptPath, installScriptPath)
	}

	log.Debug().Msgf("generated dockerfile for build context %s\n%s", buildContextPath, dockerFile)

	file, err := os.OpenFile(filepath.Join(buildContextPath, "Dockerfile"), os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return "", fmt.Errorf("failed to create Dockerfile: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(dockerFile)
	if err != nil {
		return "", fmt.Errorf("failed to write content to Dockerfile: %w", err)
	}

	return dockerFile, nil
}
