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

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/harness/gitness/gitrpc/hash"
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
	Save(filePath string, data io.Reader) (string, error)
}

type RepositoryService struct {
	rpc.UnimplementedRepositoryServiceServer
	adapter        GitAdapter
	store          Storage
	reposRoot      string
	tmpDir         string
	gitHookPath    string
	reposGraveyard string
}

func NewRepositoryService(adapter GitAdapter, store Storage, reposRoot string, tmpDir string,
	gitHookPath string, reposGraveyard string) (*RepositoryService, error) {
	return &RepositoryService{
		adapter:        adapter,
		store:          store,
		reposRoot:      reposRoot,
		tmpDir:         tmpDir,
		gitHookPath:    gitHookPath,
		reposGraveyard: reposGraveyard,
	}, nil
}

func (s RepositoryService) CreateRepository(stream rpc.RepositoryService_CreateRepositoryServer) error {
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

	committer := base.GetActor()
	if header.GetCommitter() != nil {
		committer = header.GetCommitter()
	}
	committerDate := time.Now().UTC()
	if header.GetCommitterDate() != 0 {
		committerDate = time.Unix(header.GetCommitterDate(), 0)
	}

	author := committer
	if header.GetAuthor() != nil {
		author = header.GetAuthor()
	}
	authorDate := committerDate
	if header.GetAuthorDate() != 0 {
		authorDate = time.Unix(header.GetAuthorDate(), 0)
	}

	nextFSElement := func() (*rpc.FileUpload, error) {
		m, errStream := stream.Recv()
		if errStream != nil {
			return nil, errStream
		}
		return m.GetFile(), nil
	}

	err = s.createRepositoryInternal(
		stream.Context(),
		base,
		header.GetDefaultBranch(),
		nextFSElement,
		committer,
		committerDate,
		author,
		authorDate,
	)
	if err != nil {
		return err
	}

	res := &rpc.CreateRepositoryResponse{}
	err = stream.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot send completion response: %v", err)
	}

	return nil
}

//nolint:gocognit // refactor if needed
func (s RepositoryService) createRepositoryInternal(
	ctx context.Context,
	base *rpc.WriteRequest,
	defaultBranch string,
	nextFSElement func() (*rpc.FileUpload, error),
	committer *rpc.Identity,
	committerDate time.Time,
	author *rpc.Identity,
	authorDate time.Time,
) error {
	log := log.Ctx(ctx)
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		return status.Errorf(codes.AlreadyExists, "repository exists already: %v", repoPath)
	}

	// create repository in repos folder
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := s.adapter.InitRepository(ctx, repoPath, true)
	// delete repo dir on error
	defer func() {
		if err != nil {
			cleanuperr := s.DeleteRepositoryBestEffort(ctx, base.GetRepoUid())
			if cleanuperr != nil {
				log.Warn().Err(cleanuperr).Msg("failed to cleanup repo dir")
			}
		}
	}()

	if err != nil {
		return processGitErrorf(err, "failed to initialize the repository")
	}

	// update default branch (currently set to non-existent branch)
	err = s.adapter.SetDefaultBranch(ctx, repoPath, defaultBranch, true)
	if err != nil {
		return processGitErrorf(err, "error updating default branch for repo '%s'", base.GetRepoUid())
	}

	// only execute file creation logic if files are provided
	//nolint: nestif
	if nextFSElement != nil {
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
			filePath, err = s.handleFileUploadIfAvailable(ctx, tempDir, nextFSElement)
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
			if committer == nil {
				committer = base.GetActor()
			}
			if author == nil {
				author = committer
			}
			// NOTE: This creates the branch in origin repo (as it doesn't exist as of now)
			// TODO: this should at least be a constant and not hardcoded?
			if err = s.addFilesAndPush(ctx, tempDir, filePaths, "HEAD:"+defaultBranch, nil, author, authorDate,
				committer, committerDate, "origin", "initial commit"); err != nil {
				return err
			}
		}
	}

	// setup server hook symlinks pointing to configured server hook binary
	// IMPORTANT: Setup hooks after repo creation to avoid issues with externally dependent services.
	for _, hook := range gitServerHookNames {
		hookPath := path.Join(repoPath, gitHooksDir, hook)
		err = os.Symlink(s.gitHookPath, hookPath)
		if err != nil {
			return status.Errorf(codes.Internal,
				"failed to setup symlink for hook '%s' ('%s' -> '%s'): %s", hook, hookPath, s.gitHookPath, err)
		}
	}

	log.Info().Msgf("repository created. Path: %s", repoPath)
	return nil
}

// isValidGitSHA returns true iff the provided string is a valid git sha (short or long form).
func isValidGitSHA(sha string) bool {
	return gitSHARegex.MatchString(sha)
}

func (s RepositoryService) DeleteRepository(
	ctx context.Context,
	request *rpc.DeleteRepositoryRequest,
) (*rpc.DeleteRepositoryResponse, error) {
	base := request.GetBase()

	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.RepoUid)

	if _, err := os.Stat(repoPath); err != nil && os.IsNotExist(err) {
		return nil, ErrNotFound(err)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check the status of the repository %v: %w", repoPath, err)
	}

	rmerr := s.DeleteRepositoryBestEffort(ctx, base.RepoUid)

	return &rpc.DeleteRepositoryResponse{}, rmerr
}

func (s *RepositoryService) DeleteRepositoryBestEffort(ctx context.Context, repoUID string) error {
	repoPath := getFullPathForRepo(s.reposRoot, repoUID)
	tempPath := path.Join(s.reposGraveyard, repoUID)

	// move current dir to a temp dir (prevent partial deletion)
	if err := os.Rename(repoPath, tempPath); err != nil {
		return fmt.Errorf("couldn't move dir %s to %s : %w", repoPath, tempPath, err)
	}

	if err := os.RemoveAll(tempPath); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to delete dir %s from graveyard", tempPath)
	}
	return nil
}

func (s RepositoryService) SyncRepository(
	ctx context.Context,
	request *rpc.SyncRepositoryRequest,
) (*rpc.SyncRepositoryResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.RepoUid)

	// create repo if requested
	_, err := os.Stat(repoPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, ErrInternalf("failed to create repo", err)
	}

	if os.IsNotExist(err) {
		if !request.CreateIfNotExists {
			return nil, ErrNotFound(err)
		}

		// the default branch doesn't matter for a sync,
		// we create an empty repo and the head will by updated as part of the Sync.
		const syncDefaultBranch = "main"
		if err = s.createRepositoryInternal(
			ctx,
			base,
			syncDefaultBranch,
			nil,
			nil,
			time.Time{},
			nil,
			time.Time{},
		); err != nil {
			return nil, err
		}
	}

	// sync repo content
	err = s.adapter.Sync(ctx, repoPath, request.GetSource(), request.GetRefSpecs())
	if err != nil {
		return nil, processGitErrorf(err, "failed to sync git repo")
	}

	// get remote default branch
	defaultBranch, err := s.adapter.GetRemoteDefaultBranch(ctx, request.GetSource())
	if errors.Is(err, types.ErrNoDefaultBranch) {
		return &rpc.SyncRepositoryResponse{
			DefaultBranch: "",
		}, nil
	}
	if err != nil {
		return nil, processGitErrorf(err, "failed to get default branch from repo")
	}

	// set default branch
	err = s.adapter.SetDefaultBranch(ctx, repoPath, defaultBranch, true)
	if err != nil {
		return nil, processGitErrorf(err, "failed to set default branch of repo")
	}

	return &rpc.SyncRepositoryResponse{
		DefaultBranch: defaultBranch,
	}, nil
}

func (s RepositoryService) HashRepository(
	ctx context.Context,
	request *rpc.HashRepositoryRequest,
) (*rpc.HashRepositoryResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.RepoUid)

	hashType, err := mapHashType(request.GetHashType())
	if err != nil {
		return nil, ErrInvalidArgumentf("unknown hash type '%s'", request.GetHashType())
	}

	aggregationType, err := mapHashAggregationType(request.GetAggregationType())
	if err != nil {
		return nil, ErrInvalidArgumentf("unknown aggregation type '%s'", request.GetAggregationType())
	}

	// add all references of the repo to the channel in a separate go routine, to allow streamed processing.
	// Ensure we cancel the go routine in case we exit the func early.
	goCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	hashChan := make(chan hash.SourceNext)

	go func() {
		// always close channel last before leaving go routine
		defer close(hashChan)
		defer func() {
			if r := recover(); r != nil {
				hashChan <- hash.SourceNext{
					Err: fmt.Errorf("panic received while filling data source: %s", debug.Stack()),
				}
			}
		}()

		// add default branch to hash
		defaultBranch, err := s.adapter.GetDefaultBranch(goCtx, repoPath)
		if err != nil {
			hashChan <- hash.SourceNext{
				Err: processGitErrorf(err, "failed to get default branch"),
			}
			return
		}

		hashChan <- hash.SourceNext{
			Data: hash.SerializeHead(defaultBranch),
		}

		err = s.adapter.WalkReferences(goCtx, repoPath, func(wre types.WalkReferencesEntry) error {
			ref, ok := wre[types.GitReferenceFieldRefName]
			if !ok {
				return errors.New("ref entry didn't contain the ref name")
			}
			sha, ok := wre[types.GitReferenceFieldObjectName]
			if !ok {
				return errors.New("ref entry didn't contain the ref object sha")
			}

			hashChan <- hash.SourceNext{
				Data: hash.SerializeReference(ref, sha),
			}

			return nil
		}, &types.WalkReferencesOptions{})
		if err != nil {
			hashChan <- hash.SourceNext{
				Err: processGitErrorf(err, "failed to walk references"),
			}
		}
	}()

	hasher, err := hash.New(hashType, aggregationType)
	if err != nil {
		return nil, ErrInternalf("failed to get new reference hasher", err)
	}
	source := hash.SourceFromChannel(ctx, hashChan)

	res, err := hasher.Hash(source)
	if err != nil {
		return nil, processGitErrorf(err, "failed to hash repository")
	}

	return &rpc.HashRepositoryResponse{
		Hash: res,
	}, nil
}
