// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxFileSize   = 1 << 20
	gitRepoSuffix = "git"

	gitReferenceNamePrefixBranch = "refs/heads/"
	gitReferenceNamePrefixTag    = "refs/tags/"

	gitHooksDir = "hooks"
)

var (
	// TODO: should be coming from caller ALWAYS.
	SystemIdentity = &rpc.Identity{
		Name:  "gitness",
		Email: "system@gitness",
	}

	gitServerHookNames = []string{
		"pre-receive",
		// "update", // update is disabled for performance reasons (called once for every ref)
		"post-receive",
	}

	// gitSHARegex defines the valid SHA format accepted by GIT (full form and short forms).
	// Note: as of now SHA is at most 40 characters long, but in the future it's moving to sha256
	// which is 64 chars - keep this forward-compatible.
	gitSHARegex = regexp.MustCompile("^[0-9a-f]{4,64}$")
)

type Storage interface {
	Save(filePath string, data bytes.Buffer) (string, error)
}

type RepositoryService struct {
	rpc.UnimplementedRepositoryServiceServer
	adapter     GitAdapter
	store       Storage
	reposRoot   string
	gitHookPath string
}

func NewRepositoryService(adapter GitAdapter, store Storage, reposRoot string,
	gitHookPath string) (*RepositoryService, error) {
	return &RepositoryService{
		adapter:     adapter,
		store:       store,
		reposRoot:   reposRoot,
		gitHookPath: gitHookPath,
	}, nil
}

//nolint:gocognit,funlen // need to refactor this code
func (s RepositoryService) CreateRepository(stream rpc.RepositoryService_CreateRepositoryServer) error {
	ctx := stream.Context()
	log := log.Ctx(ctx)

	// first get repo params from stream
	request, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "cannot receive create repository data")
	}

	header := request.GetHeader()
	if header == nil {
		return status.Errorf(codes.Internal, "expected header to be first message in stream")
	}
	log.Info().Msgf("received a create repository request %v", header)

	base := header.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())
	if _, err = os.Stat(repoPath); !os.IsNotExist(err) {
		return status.Errorf(codes.AlreadyExists, "repository exists already: %v", repoPath)
	}

	// create repository in repos folder
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err = s.adapter.InitRepository(ctx, repoPath, true)
	if err != nil {
		// on error cleanup repo dir
		if errCleanup := os.RemoveAll(repoPath); errCleanup != nil {
			log.Err(errCleanup).Msg("failed to cleanup repository dir")
		}
		return processGitErrorf(err, "failed to initialize the repository")
	}

	// update default branch (currently set to non-existent branch)
	err = s.adapter.SetDefaultBranch(ctx, repoPath, header.GetDefaultBranch(), true)
	if err != nil {
		return processGitErrorf(err, "error updating default branch for repo '%s'", base.GetRepoUid())
	}

	// we need temp dir for cloning
	tempDir, err := os.MkdirTemp("", "*-"+base.GetRepoUid())
	if err != nil {
		return fmt.Errorf("error creating temp dir for repo %s: %w", base.GetRepoUid(), err)
	}
	defer func(path string) {
		// when repo is successfully created remove temp dir
		errRm := os.RemoveAll(path)
		if errRm != nil {
			log.Err(errRm).Msg("failed to cleanup temporary dir.")
		}
	}(tempDir)

	// Clone repository to temp dir
	if err = s.adapter.Clone(ctx, repoPath, tempDir, types.CloneRepoOptions{}); err != nil {
		return processGitErrorf(err, "failed to clone repo")
	}

	// logic for receiving files
	filePaths := make([]string, 0, 16)
	for {
		var filePath string
		filePath, err = s.handleFileUploadIfAvailable(ctx, tempDir, func() (*rpc.FileUpload, error) {
			m, errStream := stream.Recv()
			if errStream != nil {
				return nil, errStream
			}
			return m.GetFile(), nil
		})
		if errors.Is(err, io.EOF) {
			log.Info().Msg("received stream EOF")
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive file: %v", err)
		}

		filePaths = append(filePaths, filePath)
	}

	if len(filePaths) > 0 {
		// NOTE: This creates the branch in origin repo (as it doesn't exist as of now)
		// TODO: this should at least be a constant and not hardcoded?
		if err = s.AddFilesAndPush(ctx, tempDir, filePaths, "HEAD:"+header.GetDefaultBranch(), SystemIdentity, SystemIdentity,
			"origin", "initial commit"); err != nil {
			return err
		}
	}

	// setup server hook symlinks pointing to configured server hook binary
	for _, hook := range gitServerHookNames {
		hookPath := path.Join(repoPath, gitHooksDir, hook)
		err = os.Symlink(s.gitHookPath, hookPath)
		if err != nil {
			return status.Errorf(codes.Internal,
				"failed to setup symlink for hook '%s' ('%s' -> '%s'): %s", hook, hookPath, s.gitHookPath, err)
		}
	}

	res := &rpc.CreateRepositoryResponse{}
	err = stream.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot send completion response: %v", err)
	}

	log.Info().Msgf("repository created. Path: %s", repoPath)
	return nil
}

// isValidGitSHA returns true iff the provided string is a valid git sha (short or long form).
func isValidGitSHA(sha string) bool {
	return gitSHARegex.MatchString(sha)
}
