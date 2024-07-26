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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/container"
)

var _ IDE = (*VSCodeWeb)(nil)

//go:embed script/install_vscode_web.sh
var installScript string

//go:embed script/find_vscode_web_path.sh
var findPathScript string

//go:embed media/vscodeweb/*
var mediaFiles embed.FS

const templateRunVSCodeWeb = "run_vscode_web.sh"
const startMarker = "START_MARKER"
const endMarker = "END_MARKER"
const rootUser = "root"
const harnessUser = "harness"

type VSCodeWebConfig struct {
	Port int
}

type VSCodeWeb struct {
	config *VSCodeWebConfig
}

func NewVsCodeWebService(config *VSCodeWebConfig) *VSCodeWeb {
	return &VSCodeWeb{config: config}
}

// Setup runs the installScript which downloads the required version of the code-server binary.
func (v *VSCodeWeb) Setup(
	ctx context.Context,
	devcontainer *devcontainer.Exec,
	_ *types.GitspaceInstance,
) ([]byte, error) {
	output := "Installing VSCode Web inside container.\n"

	_, err := devcontainer.ExecuteCommand(ctx, installScript, false, rootUser)
	if err != nil {
		return nil, fmt.Errorf("failed to install VSCode Web: %w", err)
	}

	findOutput, err := devcontainer.ExecuteCommand(ctx, findPathScript, false, rootUser)
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
func (v *VSCodeWeb) Run(ctx context.Context, devcontainer *devcontainer.Exec) ([]byte, error) {
	var output []byte

	runScript, err := template.GenerateScriptFromTemplate(
		templateRunVSCodeWeb, &template.RunVSCodeWebPayload{
			Port: strconv.Itoa(v.config.Port),
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to run VSCode Web from template %s: %w",
			templateRunVSCodeWeb,
			err,
		)
	}

	_, err = devcontainer.ExecuteCommand(ctx, runScript, true, harnessUser)
	if err != nil {
		return nil, fmt.Errorf("failed to run VSCode Web: %w", err)
	}
	return output, nil
}

// PortAndProtocol returns the port on which the code-server is listening.
func (v *VSCodeWeb) Port() int {
	return v.config.Port
}

func (v *VSCodeWeb) Type() enum.IDEType {
	return enum.IDETypeVSCodeWeb
}

func (v *VSCodeWeb) copyMediaToContainer(
	ctx context.Context,
	devcontainer *devcontainer.Exec,
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
	err = devcontainer.DockerClient.CopyToContainer(
		ctx,
		devcontainer.ContainerName,
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
