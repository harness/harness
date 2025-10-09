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

package ide

import (
	"archive/tar"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/container"
)

var _ IDE = (*VSCodeWeb)(nil)

//go:embed script/find_vscode_web_path.sh
var findPathScript string

//go:embed media/vscodeweb/*
var mediaFiles embed.FS

const templateRunVSCodeWeb = "run_vscode_web.sh"
const templateSetupVSCodeWeb = "install_vscode_web.sh"
const startMarker = "START_MARKER"
const endMarker = "END_MARKER"

type VSCodeWebConfig struct {
	Port int
}

type VSCodeWeb struct {
	urlScheme string
	config    *VSCodeWebConfig
}

func NewVsCodeWebService(config *VSCodeWebConfig, urlScheme string) *VSCodeWeb {
	return &VSCodeWeb{
		urlScheme: urlScheme,
		config:    config,
	}
}

// Setup runs the installScript which downloads the required version of the code-server binary.
func (v *VSCodeWeb) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing VSCode Web inside container...")
	payload := &gitspaceTypes.SetupVSCodeWebPayload{}
	if args != nil {
		if err := updateSetupPayloadFromArgs(args, payload, gitspaceLogger); err != nil {
			return err
		}
	}
	setupScript, err := utils.GenerateScriptFromTemplate(templateSetupVSCodeWeb, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate script to setup VSCode Web from template %s: %w",
			templateSetupVSCodeWeb,
			err,
		)
	}
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, setupScript, false, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to install VSCode Web: %w", err)
	}
	if err = v.updateMediaContent(ctx, exec); err != nil {
		return err
	}
	gitspaceLogger.Info(fmt.Sprintf("Successfully installed %s IDE inside container", enum.IDETypeVSCodeWeb))
	return nil
}

func (v *VSCodeWeb) updateMediaContent(ctx context.Context, exec *devcontainer.Exec) error {
	findOutput, err := exec.ExecuteCommand(ctx, findPathScript, true, exec.DefaultWorkingDir)
	if err != nil {
		return fmt.Errorf("failed to find VSCode Web install path: %w", err)
	}
	path := findOutput
	startIndex := strings.Index(path, startMarker)
	endIndex := strings.Index(path, endMarker)
	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return fmt.Errorf("could not find media folder path from find output: %s", path)
	}

	mediaFolderPath := path[startIndex+len(startMarker) : endIndex]
	if len(mediaFolderPath) == 0 {
		return fmt.Errorf("media folder path should not be empty, VSCode web is not installed")
	}

	err = v.copyMediaToContainer(ctx, exec, mediaFolderPath)
	if err != nil {
		return fmt.Errorf("failed to copy media folder to container at path %s: %w", mediaFolderPath, err)
	}
	return nil
}

// Run runs the code-server binary.
func (v *VSCodeWeb) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.RunVSCodeWebPayload{
		Port: strconv.Itoa(v.config.Port),
	}

	if args != nil {
		if proxyURI, exists := args[gitspaceTypes.VSCodeProxyURIArg]; exists {
			// Perform a type assertion to ensure proxyURI is a string
			proxyURIStr, ok := proxyURI.(string)
			if !ok {
				return fmt.Errorf("%s is not a string", gitspaceTypes.VSCodeProxyURIArg)
			}
			payload.ProxyURI = proxyURIStr
		}
	}
	runScript, err := utils.GenerateScriptFromTemplate(templateRunVSCodeWeb, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate script to run VSCode Web from template %s: %w",
			templateRunVSCodeWeb,
			err,
		)
	}
	gitspaceLogger.Info("Starting IDE ...")
	// Execute the script in the home directory
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, runScript, false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to run VSCode Web: %w", err)
	}
	return nil
}

func updateSetupPayloadFromArgs(
	args map[gitspaceTypes.IDEArg]interface{},
	payload *gitspaceTypes.SetupVSCodeWebPayload,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	if customization, exists := args[gitspaceTypes.VSCodeCustomizationArg]; exists {
		// Perform a type assertion to ensure customization is a VSCodeCustomizationSpecs
		vsCodeCustomizationSpecs, ok := customization.(types.VSCodeCustomizationSpecs)
		if !ok {
			return fmt.Errorf("customization is not of type VSCodeCustomizationSpecs")
		}
		payload.Extensions = vsCodeCustomizationSpecs.Extensions
		gitspaceLogger.Info(fmt.Sprintf(
			"VSCode Customizations: Extensions %v", vsCodeCustomizationSpecs.Extensions))
	}
	return nil
}

// PortAndProtocol returns the port on which the code-server is listening.
func (v *VSCodeWeb) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     v.config.Port,
		Protocol: enum.CommunicationProtocolHTTP,
	}
}

func (v *VSCodeWeb) Type() enum.IDEType {
	return enum.IDETypeVSCodeWeb
}

func (v *VSCodeWeb) copyMediaToContainer(
	ctx context.Context,
	exec *devcontainer.Exec,
	path string,
) error {
	// Create a buffer to hold the tar data
	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	// Walk through the embedded files and add them to the tar archive
	err := embedToTar(tarWriter, "media/vscodeweb", "")
	if err != nil {
		return fmt.Errorf("error creating tar archive: %w", err)
	}

	// Close the tar writer
	closeErr := tarWriter.Close()
	if closeErr != nil {
		return fmt.Errorf("error closing tar writer: %w", closeErr)
	}

	// Copy the tar archive to the container
	err = exec.DockerClient.CopyToContainer(
		ctx,
		exec.ContainerName,
		path,
		&tarBuffer,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("error copying files to container: %w", err)
	}

	return nil
}

func embedToTar(tarWriter *tar.Writer, baseDir, prefix string) error {
	entries, err := mediaFiles.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("error reading media files from base dir %s: %w", baseDir, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(baseDir, entry.Name())
		info, err2 := entry.Info()
		if err2 != nil {
			return fmt.Errorf("error getting file info for %s: %w", fullPath, err2)
		}

		// Remove the baseDir from the header name to ensure the files are copied directly into the destination
		headerName := filepath.Join(prefix, entry.Name())

		header, err2 := tar.FileInfoHeader(info, "")
		if err2 != nil {
			return fmt.Errorf("error getting file info header for %s: %w", fullPath, err2)
		}

		header.Name = strings.TrimPrefix(headerName, "/")

		if err2 = tarWriter.WriteHeader(header); err2 != nil {
			return fmt.Errorf("error writing file header %+v: %w", header, err2)
		}

		if !entry.IsDir() {
			file, err3 := mediaFiles.Open(fullPath)
			if err3 != nil {
				return fmt.Errorf("error opening file %s: %w", fullPath, err3)
			}
			defer file.Close()

			_, err3 = io.Copy(tarWriter, file)
			if err3 != nil {
				return fmt.Errorf("error copying file %s: %w", fullPath, err3)
			}
		} else {
			if err3 := embedToTar(tarWriter, fullPath, headerName); err3 != nil {
				return fmt.Errorf("error embeding file %s: %w", fullPath, err3)
			}
		}
	}

	return nil
}

// GenerateURL returns the url to redirect user to ide(here to another ta).
func (v *VSCodeWeb) GenerateURL(absoluteRepoPath, host, port, _ string) string {
	relativeRepoPath := strings.TrimPrefix(absoluteRepoPath, "/")
	ideURL := url.URL{
		Scheme:   v.urlScheme,
		Host:     host + ":" + port,
		RawQuery: filepath.Join("folder=", relativeRepoPath),
	}

	return ideURL.String()
}

func (v *VSCodeWeb) GeneratePluginURL(_, _ string) string {
	return ""
}
