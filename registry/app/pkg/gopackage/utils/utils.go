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
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	gopackagemetadata "github.com/harness/gitness/registry/app/metadata/gopackage"
	zs "github.com/harness/gitness/registry/app/pkg/commons/zipreader"

	"golang.org/x/mod/modfile"
)

// get the module name from mod file.
func GetModuleNameFromModFile(modBytes io.Reader) (string, error) {
	moduleName := ""
	scanner := bufio.NewScanner(modBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			moduleName = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning mod file: %w", err)
	}
	if moduleName == "" {
		return "", fmt.Errorf("module name not found in mod file")
	}
	return moduleName, nil
}

func getImageAndFileNameFromURL(path string) (string, string, error) {
	parts := strings.SplitN(path, "/@v/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid download path: %s", path)
	}
	image := parts[0]
	filename := parts[1]
	return image, filename, nil
}

func getVersionFromFileName(filename string) (string, error) {
	switch filename {
	case "list":
		return "", nil
	default:
		// v1.0.0.zip => v1.0.0
		ext := filepath.Ext(filename)                // e.g., ".zip"
		version := strings.TrimSuffix(filename, ext) // remove extension
		if version == "" {
			return "", fmt.Errorf("empty version in file name: %s", filename)
		}
		return version, nil
	}
}

func GetArtifactInfoFromURL(path string) (string, string, string, error) {
	var (
		image    string
		version  string
		filename string
	)
	// sample endpoint pkg/artifact-registry/go-repo/go/example.com/hello/@latest
	if strings.HasSuffix(path, "/@latest") {
		image = strings.Replace(path, "/@latest", "", 1)
		return image, "@latest", "", nil
	}

	// sample endpoint pkg/artifact-registry/go-repo/go/example.com/hello/@v/v1.0.2.zip
	// pkg/artifact-registry/go-repo/go/example.com/hello/@v/list
	image, filename, err := getImageAndFileNameFromURL(path)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get image and file name from URL: %w", err)
	}

	version, err = getVersionFromFileName(filename)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get version from file name: %w", err)
	}

	return image, version, filename, nil
}

func GetIndexFilePath(image string) string {
	return filepath.Join("/", image, "index")
}

// get the package metadata from info file.
func GetPackageMetadataFromInfoFile(infoBytes *bytes.Buffer) (gopackagemetadata.VersionMetadata, error) {
	var metadata gopackagemetadata.VersionMetadata
	if err := json.NewDecoder(infoBytes).Decode(&metadata); err != nil {
		return gopackagemetadata.VersionMetadata{}, fmt.Errorf("error decoding info file: %w", err)
	}
	return metadata, nil
}

func UpdateMetadataFromModFile(
	modBytes *bytes.Buffer, metadata *gopackagemetadata.VersionMetadata,
) error {
	data := modBytes.Bytes()
	modFile, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return fmt.Errorf("error parsing mod file: %w", err)
	}
	var deps []gopackagemetadata.Dependency
	for _, r := range modFile.Require {
		deps = append(deps, gopackagemetadata.Dependency{
			Name:    r.Mod.Path,
			Version: r.Mod.Version,
		})
	}
	metadata.Dependencies = deps
	return nil
}

func UpdateMetadataFromZipFile(
	reader io.ReadCloser, metadata *gopackagemetadata.VersionMetadata,
) error {
	zr := zs.NewReader(reader)

	for {
		header, err := zr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read zip file with error: %w", err)
		}

		if strings.HasSuffix(header.Name, "README.md") || strings.HasSuffix(header.Name, "README") {
			readme, err := parseReadme(zr)
			if err != nil {
				return fmt.Errorf("failed to parse metadata from README.md file: %w", err)
			}
			if readme != "" {
				metadata.Readme = readme
			}
		}
	}
	return nil
}

func parseReadme(f io.Reader) (readme string, err error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
