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

package git

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"

	"github.com/rs/zerolog/log"
)

const (
	// TODO: this should be configurable
	FileTransferChunkSize = 1024
)

type File struct {
	Path    string
	Content []byte
}

// TODO: this should be taken as a struct input defined in proto.
func (s *Service) addFilesAndPush(
	ctx context.Context,
	repoPath string,
	filePaths []string,
	branch string,
	env []string,
	author *Identity,
	authorDate time.Time,
	committer *Identity,
	committerDate time.Time,
	remote string,
	message string,
) error {
	if author == nil || committer == nil {
		return errors.InvalidArgument("both author and committer have to be provided")
	}

	err := s.git.AddFiles(ctx, repoPath, false, filePaths...)
	if err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}
	err = s.git.Commit(ctx, repoPath, api.CommitChangesOptions{
		Committer: api.Signature{
			Identity: api.Identity{
				Name:  committer.Name,
				Email: committer.Email,
			},
			When: committerDate,
		},
		Author: api.Signature{
			Identity: api.Identity{
				Name:  author.Name,
				Email: author.Email,
			},
			When: authorDate,
		},
		Message: message,
	})
	if err != nil {
		return fmt.Errorf("failed to commit files: %w", err)
	}

	err = s.git.Push(ctx, repoPath, api.PushOptions{
		// TODO: Don't hard-code
		Remote:  remote,
		Branch:  branch,
		Force:   false,
		Env:     env,
		Timeout: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to push files: %w", err)
	}

	return nil
}

func (s *Service) handleFileUploadIfAvailable(
	ctx context.Context,
	basePath string,
	file File,
) (string, error) {
	log := log.Ctx(ctx)

	fullPath := filepath.Join(basePath, file.Path)
	log.Info().Msgf("saving file at path %s", fullPath)
	_, err := s.store.Save(fullPath, bytes.NewReader(file.Content))
	if err != nil {
		return "", errors.Internal(err, "cannot save file to the store: %s", fullPath)
	}

	return fullPath, nil
}
