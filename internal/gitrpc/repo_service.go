// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/harness/gitness/internal/gitrpc/rpc"
)

const (
	maxFileSize    = 1 << 20
	repoSubdirName = "repos"
	gitRepoSuffix  = "git"

	gitReferenceNamePrefixBranch = "refs/heads/"
	gitReferenceNamePrefixTag    = "refs/tags/"
)

var (
	// TODO: Should be matching the sytem identity from config.
	SystemIdentity = &rpc.Identity{
		Name:  "gitness",
		Email: "system@gitness",
	}
)

type repositoryService struct {
	rpc.UnimplementedRepositoryServiceServer
	adapter   gitAdapter
	store     *localStore
	reposRoot string
}

func newRepositoryService(adapter gitAdapter, store *localStore, gitRoot string) (*repositoryService, error) {
	reposRoot := filepath.Join(gitRoot, repoSubdirName)
	if _, err := os.Stat(reposRoot); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(reposRoot, 0o700); err != nil {
			return nil, err
		}
	}

	return &repositoryService{
		adapter:   adapter,
		store:     store,
		reposRoot: reposRoot,
	}, nil
}

func (s repositoryService) getFullPathForRepo(uid string) string {
	return filepath.Join(s.reposRoot, fmt.Sprintf("%s.%s", uid, gitRepoSuffix))
}

//nolint:gocognit // need to refactor this code
func (s repositoryService) CreateRepository(stream rpc.RepositoryService_CreateRepositoryServer) error {
	// first get repo params from stream
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "cannot receive create repository data")
	}

	header := req.GetHeader()
	if header == nil {
		return status.Errorf(codes.Internal, "expected header to be first message in stream")
	}
	log.Info().Msgf("received a create repository request %v", header)

	repoPath := s.getFullPathForRepo(header.GetUid())
	if _, err = os.Stat(repoPath); !os.IsNotExist(err) {
		return status.Errorf(codes.AlreadyExists, "repository exists already: %v", repoPath)
	}

	// create repository in repos folder
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = s.adapter.InitRepository(ctx, repoPath, true)
	if err != nil {
		// on error cleanup repo dir
		defer func(path string) {
			_ = os.RemoveAll(path)
		}(repoPath)
		return fmt.Errorf("CreateRepository error: %w", err)
	}

	// update default branch (currently set to non-existent branch)
	err = s.adapter.SetDefaultBranch(ctx, repoPath, header.GetDefaultBranch(), true)
	if err != nil {
		return fmt.Errorf("error updating default branch for repo %s: %w", header.GetUid(), err)
	}

	// we need temp dir for cloning
	tempDir, err := os.MkdirTemp("", "*-"+header.GetUid())
	if err != nil {
		return fmt.Errorf("error creating temp dir for repo %s: %w", header.GetUid(), err)
	}
	defer func(path string) {
		// when repo is successfully created remove temp dir
		err2 := os.RemoveAll(path)
		if err2 != nil {
			log.Err(err2).Msg("failed to cleanup temporary dir.")
		}
	}(tempDir)

	// Clone repository to temp dir
	if err = s.adapter.Clone(ctx, repoPath, tempDir, cloneRepoOptions{}); err != nil {
		return status.Errorf(codes.Internal, "failed to clone repo: %v", err)
	}

	// logic for receiving files
	filePaths := make([]string, 0, 16)
	for {
		var filePath string
		filePath, err = s.handleFileUploadIfAvailable(tempDir, func() (*rpc.FileUpload, error) {
			m, err2 := stream.Recv()
			if err2 != nil {
				return nil, err2
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

	res := &rpc.CreateRepositoryResponse{}
	err = stream.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot send completion response: %v", err)
	}

	log.Info().Msgf("repository created. Path: %s", repoPath)
	return nil
}

func (s repositoryService) handleFileUploadIfAvailable(basePath string,
	nextFSElement func() (*rpc.FileUpload, error)) (string, error) {
	log.Info().Msg("waiting to receive file upload header")
	header, err := getFileStreamHeader(nextFSElement)
	if err != nil {
		return "", err
	}

	log.Info().Msgf("storing file at %s", header.Path)
	// work with file content chunks
	fileData := bytes.Buffer{}
	fileSize := 0
	for {
		log.Debug().Msg("waiting to receive data")

		var chunk *rpc.Chunk
		chunk, err = getFileUploadChunk(nextFSElement)
		if errors.Is(err, io.EOF) {
			// only for a header we expect a stream EOF error (for chunk its a chunk.EOF).
			return "", fmt.Errorf("data stream ended unexpectedly")
		}
		if err != nil {
			return "", err
		}

		size := len(chunk.Data)

		if size > 0 {
			log.Debug().Msgf("received a chunk with size: %d", size)

			// TODO: file size could be checked on client side?
			fileSize += size
			if fileSize > maxFileSize {
				return "", status.Errorf(codes.InvalidArgument, "file is too large: %d > %d", fileSize, maxFileSize)
			}

			// TODO: write in file as we go (instead of in buffer)
			_, err = fileData.Write(chunk.Data)
			if err != nil {
				return "", status.Errorf(codes.Internal, "cannot write chunk data: %v", err)
			}
		}

		if chunk.Eof {
			log.Info().Msg("Received file EOF")
			break
		}
	}
	fullPath := filepath.Join(basePath, header.Path)
	log.Info().Msgf("saving file at path %s", fullPath)
	_, err = s.store.Save(fullPath, fileData)
	if err != nil {
		return "", status.Errorf(codes.Internal, "cannot save file to the store: %v", err)
	}

	return fullPath, nil
}

func getFileStreamHeader(nextFileUpload func() (*rpc.FileUpload, error)) (*rpc.FileUploadHeader, error) {
	fs, err := getFileUpload(nextFileUpload)
	if err != nil {
		return nil, err
	}

	header := fs.GetHeader()
	if header == nil {
		return nil, status.Errorf(codes.Internal, "file stream is in wrong order - expected header")
	}

	return header, nil
}

func getFileUploadChunk(nextFileUpload func() (*rpc.FileUpload, error)) (*rpc.Chunk, error) {
	fs, err := getFileUpload(nextFileUpload)
	if err != nil {
		return nil, err
	}

	chunk := fs.GetChunk()
	if chunk == nil {
		return nil, status.Errorf(codes.Internal, "file stream is in wrong order - expected chunk")
	}

	return chunk, nil
}

func getFileUpload(nextFileUpload func() (*rpc.FileUpload, error)) (*rpc.FileUpload, error) {
	fs, err := nextFileUpload()
	if err != nil {
		return nil, err
	}
	if fs == nil {
		return nil, status.Errorf(codes.Internal, "file stream wasn't found")
	}
	return fs, nil
}

// TODO: this should be taken as a struct input defined in proto.
func (s repositoryService) AddFilesAndPush(
	ctx context.Context,
	repoPath string,
	filePaths []string,
	branch string,
	author *rpc.Identity,
	committer *rpc.Identity,
	remote string,
	message string,
) error {
	if author == nil || committer == nil {
		return status.Errorf(codes.InvalidArgument, "both author and committer have to be provided")
	}

	err := s.adapter.AddFiles(repoPath, false, filePaths...)
	if err != nil {
		return err
	}
	now := time.Now()
	err = s.adapter.Commit(repoPath, commitChangesOptions{
		// TODO: Add gitness signature
		committer: signature{
			identity: identity{
				name:  committer.Name,
				email: committer.Email,
			},
			when: now,
		},
		// TODO: Add gitness signature
		author: signature{
			identity: identity{
				name:  author.Name,
				email: author.Email,
			},
			when: now,
		},
		message: message,
	})
	if err != nil {
		return err
	}
	err = s.adapter.Push(ctx, repoPath, pushOptions{
		// TODO: Don't hard-code
		remote:  remote,
		branch:  branch,
		force:   false,
		mirror:  false,
		env:     nil,
		timeout: 0,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s repositoryService) GetTreeNode(ctx context.Context,
	request *rpc.GetTreeNodeRequest) (*rpc.GetTreeNodeResponse, error) {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitNode, err := s.adapter.GetTreeNode(ctx, repoPath, request.GetGitRef(), request.GetPath())
	if err != nil {
		return nil, err
	}

	res := &rpc.GetTreeNodeResponse{
		Node: &rpc.TreeNode{
			Type: mapGitNodeType(gitNode.nodeType),
			Mode: mapGitMode(gitNode.mode),
			Sha:  gitNode.sha,
			Name: gitNode.name,
			Path: gitNode.path,
		},
	}

	// TODO: improve performance, could be done in lower layer?
	if request.GetIncludeLatestCommit() {
		var commit *rpc.Commit
		commit, err = s.getLatestCommit(ctx, repoPath, request.GetGitRef(), request.GetPath())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get latest commit: %v", err)
		}
		res.Commit = commit
	}

	return res, nil
}

func (s repositoryService) GetSubmodule(ctx context.Context,
	request *rpc.GetSubmoduleRequest) (*rpc.GetSubmoduleResponse, error) {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitSubmodule, err := s.adapter.GetSubmodule(ctx, repoPath, request.GetGitRef(), request.GetPath())
	if err != nil {
		return nil, err
	}

	return &rpc.GetSubmoduleResponse{
		Submodule: &rpc.Submodule{
			Name: gitSubmodule.name,
			Url:  gitSubmodule.url,
		},
	}, nil
}

func (s repositoryService) GetBlob(ctx context.Context, request *rpc.GetBlobRequest) (*rpc.GetBlobResponse, error) {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitBlob, err := s.adapter.GetBlob(ctx, repoPath, request.GetSha(), request.GetSizeLimit())
	if err != nil {
		return nil, err
	}

	return &rpc.GetBlobResponse{
		Blob: &rpc.Blob{
			Sha:     request.GetSha(),
			Size:    gitBlob.size,
			Content: gitBlob.content,
		},
	}, nil
}

func (s repositoryService) ListTreeNodes(request *rpc.ListTreeNodesRequest,
	stream rpc.RepositoryService_ListTreeNodesServer) error {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	gitNodes, err := s.adapter.ListTreeNodes(stream.Context(), repoPath,
		request.GetGitRef(), request.GetPath(), request.GetRecursive(), request.GetIncludeLatestCommit())
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list nodes: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d nodes", len(gitNodes))

	for _, gitNode := range gitNodes {
		var commit *rpc.Commit
		if request.GetIncludeLatestCommit() {
			commit, err = mapGitCommit(gitNode.commit)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
			}
		}

		err = stream.Send(&rpc.ListTreeNodesResponse{
			Node: &rpc.TreeNode{
				Type: mapGitNodeType(gitNode.nodeType),
				Mode: mapGitMode(gitNode.mode),
				Sha:  gitNode.sha,
				Name: gitNode.name,
				Path: gitNode.path,
			},
			Commit: commit,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send node: %v", err)
		}
	}

	return nil
}

func (s repositoryService) ListCommits(request *rpc.ListCommitsRequest,
	stream rpc.RepositoryService_ListCommitsServer) error {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	gitCommits, totalCount, err := s.adapter.ListCommits(stream.Context(), repoPath, request.GetGitRef(),
		int(request.GetPage()), int(request.GetPageSize()))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list commits: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d commits (total: %d)", len(gitCommits), totalCount)

	// send info about total number of commits first
	err = stream.Send(&rpc.ListCommitsResponse{
		Data: &rpc.ListCommitsResponse_Header{
			Header: &rpc.ListCommitsResponseHeader{
				TotalCount: totalCount,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send response header: %v", err)
	}

	for i := range gitCommits {
		var commit *rpc.Commit
		commit, err = mapGitCommit(&gitCommits[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
		}

		err = stream.Send(&rpc.ListCommitsResponse{
			Data: &rpc.ListCommitsResponse_Commit{
				Commit: commit,
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send commit: %v", err)
		}
	}

	return nil
}

var listBranchesRefFields = []gitReferenceField{gitReferenceFieldRefName, gitReferenceFieldObjectName}

func (s repositoryService) ListBranches(request *rpc.ListBranchesRequest,
	stream rpc.RepositoryService_ListBranchesServer) error {
	ctx := stream.Context()
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	// get all required information from git refrences
	branches, err := s.listBranchesLoadReferenceData(ctx, repoPath, request)
	if err != nil {
		return err
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if request.GetIncludeCommit() {
		commitSHAs := make([]string, len(branches))
		for i := range branches {
			commitSHAs[i] = branches[i].Sha
		}

		var gitCommits []commit
		gitCommits, err = s.adapter.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get commits: %v", err)
		}

		for i := range gitCommits {
			branches[i].Commit, err = mapGitCommit(&gitCommits[i])
			if err != nil {
				return err
			}
		}
	}

	// send out all branches
	for _, branch := range branches {
		err = stream.Send(&rpc.ListBranchesResponse{
			Branch: branch,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send branch: %v", err)
		}
	}

	return nil
}

func (s repositoryService) listBranchesLoadReferenceData(ctx context.Context,
	repoPath string, request *rpc.ListBranchesRequest) ([]*rpc.Branch, error) {
	// TODO: can we be smarter with slice allocation
	branches := make([]*rpc.Branch, 0, 16)
	handler := listBranchesWalkReferencesHandler(&branches)
	instructor, endsAfter, err := wrapInstructorWithOptionalPagination(
		defaultInstructor, // branches only have one target type, default instructor is enough
		request.GetPage(),
		request.GetPageSize())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid pagination details: %v", err)
	}

	opts := &walkReferencesOptions{
		patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixBranch, request.GetQuery()),
		sort:       mapListBranchesSortOption(request.Sort),
		order:      mapSortOrder(request.Order),
		fields:     listBranchesRefFields,
		instructor: instructor,
		// we don't do any post-filtering, restrict git to only return as many elements as pagination needs.
		maxWalkDistance: endsAfter,
	}

	err = s.adapter.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get branch references: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d branches", len(branches))

	return branches, nil
}

func listBranchesWalkReferencesHandler(branches *[]*rpc.Branch) walkReferencesHandler {
	return func(e walkReferencesEntry) error {
		fullRefName, ok := e[gitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[gitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}

		branch := &rpc.Branch{
			Name: fullRefName[len(gitReferenceNamePrefixBranch):],
			Sha:  objectSHA,
		}

		// TODO: refactor to not use slice pointers?
		*branches = append(*branches, branch)

		return nil
	}
}

func newInstructorWithObjectTypeFilter(filter []gitObjectType) walkReferencesInstructor {
	return func(wre walkReferencesEntry) (walkInstruction, error) {
		v, ok := wre[gitReferenceFieldObjectType]
		if !ok {
			return walkInstructionStop, fmt.Errorf("ref field for object type is missing")
		}

		// only handle if any of the filters match
		for _, field := range filter {
			if v == string(field) {
				return walkInstructionHandle, nil
			}
		}

		// by default skip
		return walkInstructionSkip, nil
	}
}

// wrapInstructorWithOptionalPagination wraps the provided walkInstructor with pagination.
// If no paging is enabled, the original instructor is returned.
func wrapInstructorWithOptionalPagination(inner walkReferencesInstructor,
	page int32, pageSize int32) (walkReferencesInstructor, int32, error) {
	// ensure pagination is requested
	if pageSize < 1 {
		return inner, 0, nil
	}

	// sanitize page
	if page < 1 {
		page = 1
	}

	// ensure we don't overflow
	if int64(page)*int64(pageSize) > int64(math.MaxInt) {
		return nil, 0, fmt.Errorf("page %d with pageSize %d is out of range", page, pageSize)
	}

	startAfter := (page - 1) * pageSize
	endAfter := page * pageSize

	// we have to count ourselves for proper pagination
	c := int32(0)
	return func(e walkReferencesEntry) (walkInstruction, error) {
			// execute inner instructor
			inst, err := inner(e)
			if err != nil {
				return inst, err
			}

			// no pagination if element is filtered out
			if inst != walkInstructionHandle {
				return inst, nil
			}

			// increase count iff element is part of filtered output
			c++

			// add pagination on filtered output
			switch {
			case c <= startAfter:
				return walkInstructionSkip, nil
			case c > endAfter:
				return walkInstructionStop, nil
			default:
				return walkInstructionHandle, nil
			}
		},
		endAfter,
		nil
}

//nolint:gocognit // need to refactor this code
func (s repositoryService) ListCommitTags(request *rpc.ListCommitTagsRequest,
	stream rpc.RepositoryService_ListCommitTagsServer) error {
	ctx := stream.Context()
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	// get all required information from git references
	tags, err := s.listCommitTagsLoadReferenceData(ctx, repoPath, request)
	if err != nil {
		return err
	}

	// get all tag and commit SHAs
	annotatedTagSHAs := make([]string, 0, len(tags))
	commitSHAs := make([]string, len(tags))
	for i, tag := range tags {
		// always set the commit sha (will be overwritten for annotated tags)
		commitSHAs[i] = tag.Sha

		if tag.IsAnnotated {
			annotatedTagSHAs = append(annotatedTagSHAs, tag.Sha)
		}
	}

	if len(annotatedTagSHAs) > 0 {
		var gitTags []tag
		gitTags, err = s.adapter.GetAnnotatedTags(ctx, repoPath, annotatedTagSHAs)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get annotated tag: %v", err)
		}

		ai := 0 // since only some tags are annotated, we need second index
		for i := range tags {
			if !tags[i].IsAnnotated {
				continue
			}

			// correct the commitSHA for the annotated tag (currently it is the tag sha, not the commit sha)
			// NOTE: This is required as otherwise gitea will wrongly set the committer to the tagger signature.
			commitSHAs[i] = gitTags[ai].targetSha

			// update tag information with annotation details
			// NOTE: we keep the name from the reference and ignore the annotated name (similar to github)
			tags[i].Message = gitTags[ai].message
			tags[i].Title = gitTags[ai].title
			tags[i].Tagger = mapGitSignature(gitTags[ai].tagger)

			ai++
		}
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if request.GetIncludeCommit() {
		var gitCommits []commit
		gitCommits, err = s.adapter.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get commits: %v", err)
		}

		for i := range gitCommits {
			tags[i].Commit, err = mapGitCommit(&gitCommits[i])
			if err != nil {
				return err
			}
		}
	}

	// send out all tags
	for _, tag := range tags {
		err = stream.Send(&rpc.ListCommitTagsResponse{
			Tag: tag,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send tag: %v", err)
		}
	}

	return nil
}

var listCommitTagsRefFields = []gitReferenceField{gitReferenceFieldRefName,
	gitReferenceFieldObjectType, gitReferenceFieldObjectName}
var listCommitTagsObjectTypeFilter = []gitObjectType{gitObjectTypeCommit, gitObjectTypeTag}

func (s repositoryService) listCommitTagsLoadReferenceData(ctx context.Context,
	repoPath string, request *rpc.ListCommitTagsRequest) ([]*rpc.CommitTag, error) {
	// TODO: can we be smarter with slice allocation
	tags := make([]*rpc.CommitTag, 0, 16)
	handler := listCommitTagsWalkReferencesHandler(&tags)
	instructor, _, err := wrapInstructorWithOptionalPagination(
		newInstructorWithObjectTypeFilter(listCommitTagsObjectTypeFilter),
		request.GetPage(),
		request.GetPageSize())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid pagination details: %v", err)
	}

	opts := &walkReferencesOptions{
		patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixTag, request.GetQuery()),
		sort:       mapListCommitTagsSortOption(request.Sort),
		order:      mapSortOrder(request.Order),
		fields:     listCommitTagsRefFields,
		instructor: instructor,
		// we do post-filtering, so we can't restrict the git output ...
		maxWalkDistance: 0,
	}

	err = s.adapter.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tag references: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d tags", len(tags))

	return tags, nil
}

func listCommitTagsWalkReferencesHandler(tags *[]*rpc.CommitTag) walkReferencesHandler {
	return func(e walkReferencesEntry) error {
		fullRefName, ok := e[gitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[gitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}
		objectTypeRaw, ok := e[gitReferenceFieldObjectType]
		if !ok {
			return fmt.Errorf("entry missing object type")
		}

		tag := &rpc.CommitTag{
			Name:        fullRefName[len(gitReferenceNamePrefixTag):],
			Sha:         objectSHA,
			IsAnnotated: objectTypeRaw == string(gitObjectTypeTag),
		}

		// TODO: refactor to not use slice pointers?
		*tags = append(*tags, tag)

		return nil
	}
}

// createReferenceWalkPatternsFromQuery returns a list of patterns that
// ensure only references matching the basePath and query are part of the walk.
func createReferenceWalkPatternsFromQuery(basePath string, query string) []string {
	if basePath == "" && query == "" {
		return []string{}
	}

	// ensure non-empty basepath ends with "/" for proper matching and concatination.
	if basePath != "" && basePath[len(basePath)-1] != '/' {
		basePath += "/"
	}

	// in case query is empty, we just match the basePath.
	if query == "" {
		return []string{basePath}
	}

	// sanitze the query and get special chars
	query, matchPrefix, matchSuffix := sanitizeReferenceQuery(query)

	// In general, there are two search patterns:
	//   - refs/tags/**/*QUERY* - finds all refs that have QUERY in the filename.
	//   - refs/tags/**/*QUERY*/** - finds all refs that have a parent folder with QUERY in the name.
	//
	// In case the suffix has to match, they will be the same, so we return only one pattern.
	if matchSuffix {
		// exact match (refs/tags/QUERY)
		if matchPrefix {
			return []string{basePath + query}
		}

		// suffix only match (refs/tags/**/*QUERY)
		return []string{basePath + "**/*" + query}
	}

	// prefix only match
	//   - refs/tags/QUERY*
	//   - refs/tags/QUERY*/**
	if matchPrefix {
		return []string{
			basePath + query + "*",    // file
			basePath + query + "*/**", // folder
		}
	}

	// arbitrary match
	//   - refs/tags/**/*QUERY*
	//   - refs/tags/**/*QUERY*/**
	return []string{
		basePath + "**/*" + query + "*",    // file
		basePath + "**/*" + query + "*/**", // folder
	}
}

func mapSortOrder(s rpc.SortOrder) sortOrder {
	switch s {
	case rpc.SortOrder_Asc:
		return SortOrderAsc
	case rpc.SortOrder_Desc:
		return sortOrderDesc
	case rpc.SortOrder_Default:
		return sortOrderDefault
	default:
		// no need to error out - just use default for sorting
		return sortOrderDefault
	}
}

func mapListCommitTagsSortOption(s rpc.ListCommitTagsRequest_SortOption) gitReferenceField {
	switch s {
	case rpc.ListCommitTagsRequest_Date:
		return gitReferenceFieldCreatorDate
	case rpc.ListCommitTagsRequest_Name:
		return gitReferenceFieldRefName
	case rpc.ListCommitTagsRequest_Default:
		return gitReferenceFieldRefName
	default:
		// no need to error out - just use default for sorting
		return gitReferenceFieldRefName
	}
}

func mapListBranchesSortOption(s rpc.ListBranchesRequest_SortOption) gitReferenceField {
	switch s {
	case rpc.ListBranchesRequest_Date:
		return gitReferenceFieldCreatorDate
	case rpc.ListBranchesRequest_Name:
		return gitReferenceFieldRefName
	case rpc.ListBranchesRequest_Default:
		return gitReferenceFieldRefName
	default:
		// no need to error out - just use default for sorting
		return gitReferenceFieldRefName
	}
}

// sanitizeReferenceQuery removes characters that aren't allowd in a branch name.
// TODO: should we error out instead of ignore bad chars?
func sanitizeReferenceQuery(query string) (string, bool, bool) {
	if query == "" {
		return "", false, false
	}

	// get special characters before anything else
	matchPrefix := query[0] == '^' // will be removed by mapping
	matchSuffix := query[len(query)-1] == '$'
	if matchSuffix {
		// Special char $ has to be removed manually as it's a valid char
		// TODO: this restricts the query language to a certain degree, can we do better? (escaping)
		query = query[:len(query)-1]
	}

	// strip all unwanted characters
	return strings.Map(func(r rune) rune {
			// See https://git-scm.com/docs/git-check-ref-format#_description for more details.
			switch {
			// rule 4.
			case r < 32 || r == 127 || r == ' ' || r == '~' || r == '^' || r == ':':
				return -1

			// rule 5
			case r == '?' || r == '*' || r == '[':
				return -1

			// everything else we map as is
			default:
				return r
			}
		}, query),
		matchPrefix,
		matchSuffix
}

func (s repositoryService) getLatestCommit(ctx context.Context, repoPath string,
	ref string, path string) (*rpc.Commit, error) {
	gitCommit, err := s.adapter.GetLatestCommit(ctx, repoPath, ref, path)
	if err != nil {
		return nil, err
	}

	return mapGitCommit(gitCommit)
}

// TODO: Add UTs to ensure enum values match!
func mapGitNodeType(t treeNodeType) rpc.TreeNodeType {
	return rpc.TreeNodeType(t)
}

// TODO: Add UTs to ensure enum values match!
func mapGitMode(m treeNodeMode) rpc.TreeNodeMode {
	return rpc.TreeNodeMode(m)
}

func mapGitCommit(gitCommit *commit) (*rpc.Commit, error) {
	if gitCommit == nil {
		return nil, status.Errorf(codes.Internal, "git commit is nil")
	}

	return &rpc.Commit{
		Sha:       gitCommit.sha,
		Title:     gitCommit.title,
		Message:   gitCommit.message,
		Author:    mapGitSignature(gitCommit.author),
		Committer: mapGitSignature(gitCommit.committer),
	}, nil
}

func mapGitSignature(gitSignature signature) *rpc.Signature {
	return &rpc.Signature{
		Identity: &rpc.Identity{
			Name:  gitSignature.identity.name,
			Email: gitSignature.identity.email,
		},
		When: gitSignature.when.Unix(),
	}
}
