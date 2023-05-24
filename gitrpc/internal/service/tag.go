// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//nolint:gocognit // need to refactor this code
func (s ReferenceService) ListCommitTags(request *rpc.ListCommitTagsRequest,
	stream rpc.ReferenceService_ListCommitTagsServer) error {
	ctx := stream.Context()
	base := request.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

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
		var gitTags []types.Tag
		gitTags, err = s.adapter.GetAnnotatedTags(ctx, repoPath, annotatedTagSHAs)
		if err != nil {
			return processGitErrorf(err, "failed to get annotated tag")
		}

		ai := 0 // since only some tags are annotated, we need second index
		for i := range tags {
			if !tags[i].IsAnnotated {
				continue
			}

			// correct the commitSHA for the annotated tag (currently it is the tag sha, not the commit sha)
			// NOTE: This is required as otherwise gitea will wrongly set the committer to the tagger signature.
			commitSHAs[i] = gitTags[ai].TargetSha

			// update tag information with annotation details
			// NOTE: we keep the name from the reference and ignore the annotated name (similar to github)
			tags[i].Message = gitTags[ai].Message
			tags[i].Title = gitTags[ai].Title
			tags[i].Tagger = mapGitSignature(gitTags[ai].Tagger)

			ai++
		}
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if request.GetIncludeCommit() {
		var gitCommits []types.Commit
		gitCommits, err = s.adapter.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return processGitErrorf(err, "failed to get commits")
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

func newInstructorWithObjectTypeFilter(filter []types.GitObjectType) types.WalkReferencesInstructor {
	return func(wre types.WalkReferencesEntry) (types.WalkInstruction, error) {
		v, ok := wre[types.GitReferenceFieldObjectType]
		if !ok {
			return types.WalkInstructionStop, fmt.Errorf("ref field for object type is missing")
		}

		// only handle if any of the filters match
		for _, field := range filter {
			if v == string(field) {
				return types.WalkInstructionHandle, nil
			}
		}

		// by default skip
		return types.WalkInstructionSkip, nil
	}
}

var listCommitTagsRefFields = []types.GitReferenceField{types.GitReferenceFieldRefName,
	types.GitReferenceFieldObjectType, types.GitReferenceFieldObjectName}
var listCommitTagsObjectTypeFilter = []types.GitObjectType{types.GitObjectTypeCommit, types.GitObjectTypeTag}

func (s ReferenceService) listCommitTagsLoadReferenceData(ctx context.Context,
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

	opts := &types.WalkReferencesOptions{
		Patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixTag, request.GetQuery()),
		Sort:       mapListCommitTagsSortOption(request.Sort),
		Order:      mapSortOrder(request.Order),
		Fields:     listCommitTagsRefFields,
		Instructor: instructor,
		// we do post-filtering, so we can't restrict the git output ...
		MaxWalkDistance: 0,
	}

	err = s.adapter.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, processGitErrorf(err, "failed to walk tag references")
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d tags", len(tags))

	return tags, nil
}

func listCommitTagsWalkReferencesHandler(tags *[]*rpc.CommitTag) types.WalkReferencesHandler {
	return func(e types.WalkReferencesEntry) error {
		fullRefName, ok := e[types.GitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[types.GitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}
		objectTypeRaw, ok := e[types.GitReferenceFieldObjectType]
		if !ok {
			return fmt.Errorf("entry missing object type")
		}

		tag := &rpc.CommitTag{
			Name:        fullRefName[len(gitReferenceNamePrefixTag):],
			Sha:         objectSHA,
			IsAnnotated: objectTypeRaw == string(types.GitObjectTypeTag),
		}

		// TODO: refactor to not use slice pointers?
		*tags = append(*tags, tag)

		return nil
	}
}
func (s ReferenceService) CreateTag(
	ctx context.Context,
	request *rpc.CreateTagRequest,
) (*rpc.CreateTagResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to open repo")
	}

	sharedRepo, err := NewSharedRepo(s.tmpDir, base.GetRepoUid(), repo)
	if err != nil {
		return nil, processGitErrorf(err, "failed to create new shared repo")
	}

	defer sharedRepo.Close(ctx)

	err = sharedRepo.Clone(ctx, "")
	if err != nil {
		return nil, processGitErrorf(err, "failed to clone shared repo with branch '%s'", request.GetSha())
	}
	actor := request.GetBase().GetActor()
	createTagRequest := types.CreateTagRequest{
		Name:        request.GetTagName(),
		TargetSha:   request.GetSha(),
		Message:     request.GetMessage(),
		TaggerEmail: actor.GetEmail(),
		TaggerName:  actor.GetName(),
	}
	err = s.adapter.CreateAnnotatedTag(ctx, sharedRepo.tmpPath, &createTagRequest)

	if err != nil {
		return nil, processGitErrorf(err, "Failed to create tag %s - %s", request.GetTagName(), err.Error())
	}

	if err = sharedRepo.PushTag(ctx, base, request.GetTagName()); err != nil {
		return nil, processGitErrorf(err, "Failed to push the tag %s to remote", request.GetTagName())
	}

	tag, err := s.adapter.GetAnnotatedTag(ctx, repoPath, request.GetTagName())

	if err != nil {
		return nil, err
	}
	commitTag := mapCommitTag(tag)
	return &rpc.CreateTagResponse{Tag: commitTag}, nil
}

func (s ReferenceService) DeleteTag(
	ctx context.Context,
	request *rpc.DeleteTagRequest,
) (*rpc.UpdateRefResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to open repo")
	}

	sharedRepo, err := NewSharedRepo(s.tmpDir, base.GetRepoUid(), repo)
	if err != nil {
		return nil, processGitErrorf(err, "failed to create new shared repo")
	}

	defer sharedRepo.Close(ctx)

	err = sharedRepo.Clone(ctx, request.GetTagName())
	if err != nil {
		return nil, processGitErrorf(err, "failed to clone shared repo with tag '%s'", request.GetTagName())
	}

	if err = sharedRepo.PushDeleteTag(ctx, base, request.GetTagName()); err != nil {
		return nil, processGitErrorf(err, "Failed to push the tag %s to remote", request.GetTagName())
	}

	return &rpc.UpdateRefResponse{}, nil
}
