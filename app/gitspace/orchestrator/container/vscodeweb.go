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

package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	dockerTypes "github.com/docker/docker/api/types"

	_ "embed"
)

var _ IDE = (*VSCodeWeb)(nil)

//go:embed script/run_vscode_web.sh
var runScript string

//go:embed script/find_vscode_web_path.sh
var findPathScript string

const templateInstallVSCodeWeb = "install_vscode_web.sh"
const startMarker = "START_MARKER"
const endMarker = "END_MARKER"

type VSCodeWebConfig struct {
	Port string
}

type VSCodeWeb struct {
	config *VSCodeWebConfig
}

func NewVsCodeWebService(config *VSCodeWebConfig) *VSCodeWeb {
	return &VSCodeWeb{config: config}
}

// Setup runs the installScript which downloads the required version of the code-server binary.
func (v *VSCodeWeb) Setup(ctx context.Context, devcontainer *Devcontainer, _ *types.GitspaceInstance) ([]byte, error) {
	installScript, err := GenerateScriptFromTemplate(
		templateInstallVSCodeWeb, &InstallVSCodeWebPayload{
			Port: v.config.Port,
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to install VSCode Web from template %s: %w",
			templateInstallVSCodeWeb,
			err,
		)
	}

	output := "Installing VSCode Web inside container.\n"

	_, err = devcontainer.ExecuteCommand(ctx, installScript, false)
	if err != nil {
		return nil, fmt.Errorf("failed to install VSCode Web: %w", err)
	}

	findOutput, err := devcontainer.ExecuteCommand(ctx, findPathScript, false)
	if err != nil {
		return nil, fmt.Errorf("failed to find VSCode Web install path: %w", err)
	}

	path := string(findOutput)
	startIndex := strings.Index(path, startMarker)
	endIndex := strings.Index(path, endMarker)
	if startIndex == -1 || endIndex == -1 || startIndex >= endIndex {
		return nil, fmt.Errorf("could not find media folder path from find output: %s", path)
	}

	mediaFolderPath := path[startIndex+len(startMarker) : endIndex]
	err = v.copyMediaToContainer(ctx, devcontainer, mediaFolderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy media folder to container at path %s: %w", mediaFolderPath, err)
	}

	output += "Successfully installed VSCode Web inside container.\n"

	return []byte(output), nil
}

// Run runs the code-server binary.
func (v *VSCodeWeb) Run(ctx context.Context, devcontainer *Devcontainer) ([]byte, error) {
	var output []byte

	_, err := devcontainer.ExecuteCommand(ctx, runScript, true)
	if err != nil {
		return nil, fmt.Errorf("failed to run VSCode Web: %w", err)
	}
	return output, nil
}

// PortAndProtocol returns the port on which the code-server is listening.
func (v *VSCodeWeb) PortAndProtocol() string {
	return v.config.Port + "/tcp"
}

func (v *VSCodeWeb) Type() enum.IDEType {
	return enum.IDETypeVSCodeWeb
}

func (v *VSCodeWeb) copyMediaToContainer(ctx context.Context, devcontainer *Devcontainer, path string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not find working directory cwd: %w", err)
	}
	// TODO: Can this be decoupled from the project structure?
	mediaDir := filepath.Join(cwd, "app", "gitspace", "orchestrator", "container", "media", "vscodeweb")

	// Create a buffer to hold the tar data
	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	// Walk through the source directory and add files to the tar archive
	err = filepath.Walk(mediaDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a tar header for each file
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		x, err := filepath.Rel(mediaDir, filePath)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(x) // Relative path for tar header

		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			// If it's a file, write its contents
			file, fileErr := os.Open(filePath)
			if fileErr != nil {
				return fileErr
			}

			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error creating tar archive: %w", err)
	}

	// Close the tar writer
	closeErr := tarWriter.Close()
	if closeErr != nil {
		return fmt.Errorf("error closing tar writer: %w", closeErr)
	}

	// Copy the tar archive to the container
	err = devcontainer.DockerClient.CopyToContainer(
		ctx,
		devcontainer.ContainerName,
		path,
		&tarBuffer,
		dockerTypes.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("error copying files to container: %w", err)
	}

	return nil
}
