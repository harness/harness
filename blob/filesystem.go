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

package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	fileDiskPathFmt = "%s/%s"
)

type FileSystemStore struct {
	basePath string
}

func NewFileSystemStore(cfg Config) (Store, error) {
	return &FileSystemStore{
		basePath: cfg.Bucket,
	}, nil
}

func (c FileSystemStore) Upload(ctx context.Context,
	file io.Reader,
	filePath string,
) error {
	fileDiskPath := fmt.Sprintf(fileDiskPathFmt, c.basePath, filePath)

	dir, _ := path.Split(fileDiskPath)
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(dir, os.ModeDir|os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create parent directory for the file: %w", err)
		}
	}

	destinationFile, err := os.Create(fileDiskPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		cErr := destinationFile.Close()
		if cErr != nil {
			log.Ctx(ctx).Warn().Err(cErr).
				Msgf("failed to close destination file %q in directory %q", filePath, c.basePath)
		}
	}()

	if _, err := io.Copy(destinationFile, file); err != nil {
		// Remove the file if it was created.
		removeErr := os.Remove(fileDiskPath)
		if removeErr != nil {
			// Best effort attempt to remove the file on write failure.
			log.Ctx(ctx).Warn().Err(removeErr).Msgf(
				"failed to cleanup file %q in directory %q after write to filesystem failed with %s",
				filePath, c.basePath, err)
		}
		return fmt.Errorf("failed to write file to filesystem: %w", err)
	}

	return nil
}

func (c *FileSystemStore) GetSignedURL(
	context.Context,
	string,
	time.Time,
	...SignURLOption) (string, error) {
	return "", ErrNotSupported
}

func (c *FileSystemStore) Download(_ context.Context, filePath string) (io.ReadCloser, error) {
	fileDiskPath := fmt.Sprintf(fileDiskPathFmt, c.basePath, filePath)

	file, err := os.Open(fileDiskPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return io.ReadCloser(file), nil
}

func (c *FileSystemStore) Move(_ context.Context, srcPath, dstPath string) error {
	srcDiskPath := fmt.Sprintf(fileDiskPathFmt, c.basePath, srcPath)
	dstDiskPath := fmt.Sprintf(fileDiskPathFmt, c.basePath, dstPath)

	// Ensure destination directory exists
	dstDir, _ := path.Split(dstDiskPath)
	if _, err := os.Stat(dstDir); errors.Is(err, fs.ErrNotExist) {
		if err = os.MkdirAll(dstDir, os.ModeDir|os.ModePerm); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	if err := os.Rename(srcDiskPath, dstDiskPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}
	return nil
}

func (c *FileSystemStore) Delete(_ context.Context, filePath string) error {
	fileDiskPath := fmt.Sprintf(fileDiskPathFmt, c.basePath, filePath)

	if err := os.Remove(fileDiskPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
