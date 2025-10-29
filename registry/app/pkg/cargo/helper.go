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

package cargo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"

	"github.com/pelletier/go-toml/v2"
)

func getCrateFileName(imageName string, version string) string {
	return fmt.Sprintf("%s-%s.crate", imageName, version)
}

func getCrateFilePath(imageName string, version string) string {
	return fmt.Sprintf("/crates/%s/%s/%s", imageName, version, getCrateFileName(imageName, version))
}

func getPackageIndexFilePath(filePath string) string {
	return fmt.Sprintf("/%s/%s", "index", filePath)
}

func downloadPackageFilePath(imageName string, version string) string {
	base, _ := url.Parse("")
	segments := []string{base.Path}
	segments = append(segments, imageName)
	segments = append(segments, version)
	segments = append(segments, "download")
	base.Path = path.Join(segments...)
	return strings.TrimRight(base.String(), "/")
}

type cargoManifest struct {
	Package struct {
		Name          string   `toml:"name"`
		Version       string   `toml:"version"`
		Description   string   `toml:"description"`
		Edition       string   `toml:"edition"`
		License       string   `toml:"license"`
		Keywords      []string `toml:"keywords"`
		Repository    string   `toml:"repository"`
		Documentation string   `toml:"documentation"`
		Readme        any      `toml:"readme"` // value can be bool or string. so normalising this later in the code
		ReadmeContent string
	} `toml:"package"`

	Dependencies      map[string]any `toml:"dependencies"`
	DevDependencies   map[string]any `toml:"dev-dependencies"`
	BuildDependencies map[string]any `toml:"build-dependencies"`
}

func getReadmePath(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return "README.md" // conventionally true implies README.md
		}
		return ""
	default:
		return ""
	}
}

func generateMetadataFromFile(
	info *cargotype.ArtifactInfo, fileReader *storage.FileReader,
) (*cargometadata.VersionMetadata, error) {
	files, err := extractCrate(fileReader)
	if err != nil {
		return nil, err
	}

	folderPath := fmt.Sprintf("%s-%s", info.Image, info.Version)
	tomlFilePath := fmt.Sprintf("%s/Cargo.toml", folderPath)
	cargoTomlData, ok := files[tomlFilePath]
	if !ok {
		return nil, fmt.Errorf("cargo.toml not found in crate")
	}

	metadata, err := parseCargoToml(cargoTomlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cargo.toml: %w", err)
	}
	metadata.Package.ReadmeContent = ""
	readmePath := getReadmePath(metadata.Package.Readme)
	if readmePath != "" {
		readmeFilePath := fmt.Sprintf("%s/%s", folderPath, readmePath)
		readmeData, ok := files[readmeFilePath]
		if !ok {
			return nil, fmt.Errorf("readme not found in crate %s: %w", readmeFilePath, err)
		}
		metadata.Package.ReadmeContent = string(readmeData)
	}

	return mapCargoManifestToVersionMetadata(metadata), nil
}

func extractCrate(file io.ReadCloser) (map[string][]byte, error) {
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	files := make(map[string][]byte)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			files[hdr.Name] = content
		}
	}

	return files, nil
}

func parseCargoToml(data []byte) (*cargoManifest, error) {
	var manifest cargoManifest
	err := toml.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

func mapCargoManifestToVersionMetadata(manifest *cargoManifest) *cargometadata.VersionMetadata {
	return &cargometadata.VersionMetadata{
		Name:             manifest.Package.Name,
		Version:          manifest.Package.Version,
		Description:      manifest.Package.Description,
		License:          manifest.Package.License,
		Keywords:         manifest.Package.Keywords,
		RepositoryURL:    manifest.Package.Repository,
		DocumentationURL: manifest.Package.Documentation,
		Readme:           manifest.Package.ReadmeContent,
		Yanked:           false,
		Dependencies:     *parseDependencies(manifest),
	}
}

func parseDependencies(manifest *cargoManifest) *[]cargometadata.VersionDependency {
	var deps []cargometadata.VersionDependency

	// Regular dependencies
	for name, raw := range manifest.Dependencies {
		dep := parseDependency(name, raw, cargometadata.DependencyKindTypeNormal, "")
		deps = append(deps, dep)
	}

	// Dev dependencies
	for name, raw := range manifest.DevDependencies {
		dep := parseDependency(name, raw, cargometadata.DependencyKindTypeDev, "")
		deps = append(deps, dep)
	}

	// Build dependencies
	for name, raw := range manifest.BuildDependencies {
		dep := parseDependency(name, raw, cargometadata.DependencyKindTypeBuild, "")
		deps = append(deps, dep)
	}
	return &deps
}

func parseDependency(
	name string, raw any, kind cargometadata.DependencyKindType,
	target string,
) cargometadata.VersionDependency {
	dep := cargometadata.VersionDependency{
		Dependency: cargometadata.Dependency{
			Name:               name,
			Kind:               kind,
			Target:             target,
			ExplicitNameInToml: name,
			DefaultFeatures:    true, // default in Cargo
		},
		VersionRequired: "",
	}

	switch val := raw.(type) {
	case string:
		// Simple version
		dep.VersionRequired = val
	case map[string]any:
		if version, ok := val["version"].(string); ok {
			dep.VersionRequired = version
		}
		if features, ok := val["features"].([]any); ok {
			for _, f := range features {
				dep.Features = append(dep.Features, fmt.Sprint(f))
			}
		}
		if optional, ok := val["optional"].(bool); ok {
			dep.IsOptional = optional
		}
		if defaultFeatures, ok := val["default-features"].(bool); ok {
			dep.DefaultFeatures = defaultFeatures
		}
		if registry, ok := val["registry"].(string); ok {
			dep.Registry = registry
		}
	}

	return dep
}
