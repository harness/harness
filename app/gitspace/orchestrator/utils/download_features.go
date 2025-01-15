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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/tidwall/jsonc"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

type featureSource struct {
	sourceURL  string
	sourceType enum.FeatureSourceType
}

// DownloadFeatures downloads the user specified features and all the features on which the user defined features
// depend. It does it by checking the dependencies of a downloaded feature from the devcontainer-feature.json
// and adding that feature to the download queue, if it is not already marked for download.
func DownloadFeatures(
	ctx context.Context,
	gitspaceInstanceIdentifier string,
	features types.Features,
) (*map[string]*types.DownloadedFeature, error) {
	downloadedFeatures := sync.Map{}
	featuresToBeDownloaded := sync.Map{}
	downloadQueue := make(chan featureSource, 100)
	errorCh := make(chan error, 100)

	// startCh and endCh are used to check if all the goroutines spawned to download features have completed or not.
	// Whenever a new goroutine is spawned, it increments the counter listening to the startCh.
	// Upon completion, it increments the counter listening to the endCh.
	// When the start count == end count and end count is > 0, it means all the goroutines have completed execution.
	startCh := make(chan int, 100)
	endCh := make(chan int, 100)

	for key, value := range features {
		featuresToBeDownloaded.Store(key, value)
		downloadQueue <- featureSource{sourceURL: key, sourceType: value.SourceType}
	}

	// NOTE: The following logic might see performance issues with spikes in memory and CPU usage.
	// If there are such issues, we can introduce throttling on the basis of memory, CPU, etc.
	go func(ctx context.Context) {
		for source := range downloadQueue {
			select {
			case <-ctx.Done():
				return
			default:
				startCh <- 1
				go func(source featureSource) {
					defer func(endCh chan int) { endCh <- 1 }(endCh)
					err := downloadFeature(ctx, gitspaceInstanceIdentifier, &source, &featuresToBeDownloaded,
						downloadQueue, &downloadedFeatures)
					errorCh <- err
				}(source)
			}
		}
	}(ctx)

	var totalStart int
	var totalEnd int
	var downloadError error
waitLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case start := <-startCh:
			totalStart += start
		case end := <-endCh:
			totalEnd += end
		case err := <-errorCh:
			if err == nil {
				continue
			}
			if downloadError != nil {
				downloadError = fmt.Errorf("error downloading features: %w\n%w", err, downloadError)
			} else {
				downloadError = fmt.Errorf("error downloading features: %w", err)
			}

		default:
			if totalEnd > 0 && totalStart == totalEnd {
				break waitLoop
			} else {
				time.Sleep(time.Millisecond * 10)
			}
		}
	}

	close(startCh)
	close(endCh)
	close(downloadQueue)
	close(errorCh)

	if downloadError != nil {
		return nil, downloadError
	}

	var finalMap = make(map[string]*types.DownloadedFeature)

	var saveToMapFunc = func(key, value any) bool {
		finalMap[key.(string)] = value.(*types.DownloadedFeature) // nolint:errcheck
		return true
	}

	downloadedFeatures.Range(saveToMapFunc)
	return &finalMap, nil
}

// downloadFeature downloads a single feature. Depending on the source type, it either downloads it from an OCI
// repo or an HTTP(S) URL. It then fetches its devcontainer-feature.json and also checks which dependencies need
// to be downloaded for this feature.
func downloadFeature(
	ctx context.Context,
	gitspaceInstanceIdentifier string,
	source *featureSource,
	featuresToBeDownloaded *sync.Map,
	downloadQueue chan featureSource,
	downloadedFeatures *sync.Map,
) error {
	var tarballName string
	var featureFolderName string
	var featureTag string
	var sourceWithoutTag string
	var canonicalName string
	var downloadDirectory string
	switch source.sourceType {
	case enum.FeatureSourceTypeOCI:
		sourceWithoutTag = strings.SplitN(source.sourceURL, ":", 2)[0]
		partialFeatureNameWithTag := filepath.Base(source.sourceURL)
		parts := strings.SplitN(partialFeatureNameWithTag, ":", 2)
		tarballName = fmt.Sprintf("devcontainer-feature-%s.tgz", parts[0])
		featureFolderName = strings.TrimSuffix(tarballName, ".tgz")
		featureTag = parts[1]
		downloadDirectory = getFeatureDownloadDirectory(gitspaceInstanceIdentifier, featureFolderName, featureTag)
		contentDigest, err := downloadTarballFromOCIRepo(ctx, source.sourceURL, downloadDirectory)
		if err != nil {
			return fmt.Errorf("error downloading oci artifact for feature %s: %w", source.sourceURL, err)
		}
		canonicalName = contentDigest

	case enum.FeatureSourceTypeTarball:
		sourceWithoutTag = source.sourceURL
		canonicalName = source.sourceURL
		featureTag = types.FeatureDefaultTag // using static tag for comparison
		tarballURL, err := url.Parse(source.sourceURL)
		if err != nil {
			return fmt.Errorf("error parsing feature URL for feature %s: %w", source.sourceURL, err)
		}
		tarballName = filepath.Base(tarballURL.Path)
		featureFolderName = strings.TrimSuffix(tarballName, ".tgz")
		downloadDirectory = getFeatureDownloadDirectory(gitspaceInstanceIdentifier, featureFolderName, featureTag)
		err = downloadTarball(source.sourceURL, downloadDirectory, filepath.Join(downloadDirectory, tarballName))
		if err != nil {
			return fmt.Errorf("error downloading tarball for feature %s: %w", source.sourceURL, err)
		}
	default:
		return fmt.Errorf("unsupported feature type: %s", source.sourceType)
	}

	devcontainerFeature, err := getDevcontainerFeatureConfig(downloadDirectory, featureFolderName, tarballName, source)
	if err != nil {
		return err
	}

	downloadedFeature := types.DownloadedFeature{
		FeatureFolderName:         featureFolderName,
		Source:                    source.sourceURL,
		SourceWithoutTag:          sourceWithoutTag,
		Tag:                       featureTag,
		CanonicalName:             canonicalName,
		DevcontainerFeatureConfig: devcontainerFeature,
	}

	downloadedFeatures.Store(source.sourceURL, &downloadedFeature)

	// Check all the dependencies which are required for this feature. If any, check if they are already marked
	// for downloaded. If not, push to the download queue.
	if downloadedFeature.DevcontainerFeatureConfig.DependsOn != nil &&
		len(*downloadedFeature.DevcontainerFeatureConfig.DependsOn) > 0 {
		for key, value := range *downloadedFeature.DevcontainerFeatureConfig.DependsOn {
			_, present := featuresToBeDownloaded.LoadOrStore(key, value)
			if !present {
				downloadQueue <- featureSource{sourceURL: key, sourceType: value.SourceType}
			}
		}
	}

	return nil
}

// getDevcontainerFeatureConfig returns the devcontainer-feature.json by unpacking the downloaded tarball,
// unmarshalling the file contents to types.DevcontainerFeatureConfig. It removes any comments before unmarshalling.
func getDevcontainerFeatureConfig(
	downloadDirectory string,
	featureName string,
	tarballName string,
	source *featureSource,
) (*types.DevcontainerFeatureConfig, error) {
	dst := filepath.Join(downloadDirectory, featureName)
	err := unpackTarball(filepath.Join(downloadDirectory, tarballName), dst)
	if err != nil {
		return nil, fmt.Errorf("error unpacking tarball for feature %s: %w", source.sourceURL, err)
	}

	// Delete the tarball to avoid unnecessary packaging and copying during docker build.
	err = os.Remove(filepath.Join(downloadDirectory, tarballName))
	if err != nil {
		return nil, fmt.Errorf("error deleting tarball for feature %s: %w", source.sourceURL, err)
	}

	devcontainerFeatureRaw, err := os.ReadFile(filepath.Join(dst, "devcontainer-feature.json"))
	if err != nil {
		return nil, fmt.Errorf("error reading devcontainer-feature.json file for feature %s: %w",
			source.sourceURL, err)
	}
	var devcontainerFeature types.DevcontainerFeatureConfig
	err = json.Unmarshal(jsonc.ToJSON(devcontainerFeatureRaw), &devcontainerFeature)
	if err != nil {
		return nil, fmt.Errorf("error parsing devcontainer-feature.json for feature %s: %w",
			source.sourceURL, err)
	}
	return &devcontainerFeature, nil
}

func getGitspaceInstanceDirectory(gitspaceInstanceIdentifier string) string {
	return filepath.Join("/tmp", gitspaceInstanceIdentifier)
}

func getFeaturesDownloadDirectory(gitspaceInstanceIdentifier string) string {
	return filepath.Join(getGitspaceInstanceDirectory(gitspaceInstanceIdentifier), "devcontainer-features")
}

func getFeatureDownloadDirectory(gitspaceInstanceIdentifier, featureFolderName, featureTag string) string {
	return filepath.Join(getFeaturesDownloadDirectory(gitspaceInstanceIdentifier),
		getFeatureFolderNameWithTag(featureFolderName, featureTag))
}

func getFeatureFolderNameWithTag(featureFolderName string, featureTag string) string {
	return featureFolderName + "-" + featureTag
}

func downloadTarballFromOCIRepo(ctx context.Context, ociRepo string, filePath string) (string, error) {
	parts := strings.SplitN(ociRepo, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid oci repo: %s", ociRepo)
	}
	fs, err := file.New(filePath)
	if err != nil {
		return "", err
	}
	defer fs.Close()
	repo, err := remote.NewRepository(parts[0])
	if err != nil {
		return "", err
	}
	tag := parts[1]
	md, err := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if err != nil {
		return "", err
	}
	return md.Digest.String(), nil
}

func downloadTarball(url, dirPath, fileName string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, fs.ModePerm)
		if err != nil {
			return err
		}
	}
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url) // nolint:gosec,noctx
	if err != nil {
		return fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	// Check the HTTP response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download tarball: HTTP status %d", resp.StatusCode)
	}

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save tarball: %w", err)
	}
	return nil
}

// unpackTarball extracts a .tgz file to a specified output directory.
func unpackTarball(tarball, outputDir string) error {
	// Open the tarball file
	file, err := os.Open(tarball)
	if err != nil {
		return fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	// Create a tar reader
	tarReader := tar.NewReader(file)

	// Iterate through the files in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of tar archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Determine the file's full path
		targetPath := filepath.Join(outputDir, header.Name) // nolint:gosec

		switch header.Typeflag {
		case tar.TypeDir:
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Extract the file
			if err := extractFile(tarReader, targetPath, header.Mode); err != nil {
				return err
			}
		default:
			return fmt.Errorf("skipping unsupported type: %c in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

// extractFile writes the content of a file from the tar archive.
func extractFile(tarReader *tar.Reader, targetPath string, mode int64) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := os.Chmod(targetPath, os.FileMode(mode)); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
