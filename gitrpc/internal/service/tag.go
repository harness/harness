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
	"strings"
	"time"

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

	// populate annotation data for all annotated tags
	if len(annotatedTagSHAs) > 0 {
		var aTags []types.Tag
		aTags, err = s.adapter.GetAnnotatedTags(ctx, repoPath, annotatedTagSHAs)
		if err != nil {
			return processGitErrorf(err, "failed to get annotated tag")
		}

		ai := 0 // index for annotated tags
		ri := 0 // read index for all tags
		wi := 0 // write index for all tags (as we might remove some non-commit tags)
		for ; ri < len(tags); ri++ {
			// always copy the current read element to the latest write position (doesn't mean it's kept)
			tags[wi] = tags[ri]
			commitSHAs[wi] = commitSHAs[ri]

			// keep the tag as is if it's not annotated
			if !tags[ri].IsAnnotated {
				wi++
				continue
			}

			// filter out annotated tags that don't point to commit objects (blobs, trees, nested tags, ...)
			// we don't actually wanna write it, so keep write index
			// TODO: Support proper pagination: https://harness.atlassian.net/browse/CODE-669
			if aTags[ai].TargetType != types.GitObjectTypeCommit {
				ai++
				continue
			}

			// correct the commitSHA for the annotated tag (currently it is the tag sha, not the commit sha)
			// NOTE: This is required as otherwise gitea will wrongly set the committer to the tagger signature.
			commitSHAs[wi] = aTags[ai].TargetSha

			// update tag information with annotation details
			// NOTE: we keep the name from the reference and ignore the annotated name (similar to github)
			tags[wi].Message = aTags[ai].Message
			tags[wi].Title = aTags[ai].Title
			tags[wi].Tagger = mapGitSignature(aTags[ai].Tagger)

			ai++
			wi++
		}

		// truncate slices based on what was removed
		tags = tags[:wi]
		commitSHAs = commitSHAs[:wi]
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
func (s ReferenceService) CreateCommitTag(
	ctx context.Context,
	request *rpc.CreateCommitTagRequest,
) (*rpc.CreateCommitTagResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	sharedRepo, err := NewSharedRepo(s.tmpDir, base.GetRepoUid(), repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to create new shared repo")
	}

	defer sharedRepo.Close(ctx)

	// clone repo (with HEAD branch - target might be anything)
	err = sharedRepo.Clone(ctx, "")
	if err != nil {
		return nil, processGitErrorf(err, "failed to clone shared repo")
	}

	// get target commit (as target could be branch/tag/commit, and tag can't be pushed using source:destination syntax)
	// NOTE: in case the target is an annotated tag, the targetCommit title and message are that of the tag, not the commit
	targetCommit, err := s.adapter.GetCommit(ctx, sharedRepo.tmpPath, strings.TrimSpace(request.GetTarget()))
	if git.IsErrNotExist(err) {
		return nil, ErrNotFoundf("target '%s' doesn't exist", request.GetTarget())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get commit id for target '%s': %w", request.GetTarget(), err)
	}

	tagger := base.GetActor()
	if request.GetTagger() != nil {
		tagger = request.GetTagger()
	}
	taggerDate := time.Now().UTC()
	if request.GetTaggerDate() != 0 {
		taggerDate = time.Unix(request.GetTaggerDate(), 0)
	}

	createTagRequest := &types.CreateTagOptions{
		Message: request.GetMessage(),
		Tagger: types.Signature{
			Identity: types.Identity{
				Name:  tagger.Name,
				Email: tagger.Email,
			},
			When: taggerDate,
		},
	}
	err = s.adapter.CreateTag(
		ctx,
		sharedRepo.tmpPath,
		request.GetTagName(),
		targetCommit.SHA,
		createTagRequest)
	if errors.Is(err, types.ErrAlreadyExists) {
		return nil, ErrAlreadyExistsf("tag '%s' already exists", request.GetTagName())
	}
	if err != nil {
		return nil, processGitErrorf(err, "Failed to create tag '%s'", request.GetTagName())
	}

	if err = sharedRepo.PushTag(ctx, base, request.GetTagName()); err != nil {
		return nil, processGitErrorf(err, "Failed to push the tag to remote")
	}

	var commitTag *rpc.CommitTag
	if request.GetMessage() != "" {
		tag, err := s.adapter.GetAnnotatedTag(ctx, repoPath, request.GetTagName())
		if err != nil {
			return nil, fmt.Errorf("failed to read annotated tag after creation: %w", err)
		}
		commitTag = mapAnnotatedTag(tag)
	} else {
		commitTag = &rpc.CommitTag{
			Name:        request.GetTagName(),
			IsAnnotated: false,
			Sha:         targetCommit.SHA,
		}
	}

	// gitea overwrites some commit details in case getCommit(ref) was called with ref being a tag
	// To avoid this issue, let's get the commit again using the actual id of the commit
	// TODO: can we do this nicer?
	rawCommit, err := s.adapter.GetCommit(ctx, repoPath, targetCommit.SHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get the raw commit '%s' after tag creation: %w", targetCommit.SHA, err)
	}

	commitTag.Commit, err = mapGitCommit(rawCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to map target commit after tag creation: %w", err)
	}

	return &rpc.CreateCommitTagResponse{Tag: commitTag}, nil
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

	sharedRepo, err := NewSharedRepo(s.tmpDir, base.GetRepoUid(), repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to create new shared repo")
	}

	defer sharedRepo.Close(ctx)

	// clone repo (with HEAD branch - tag target might be anything)
	err = sharedRepo.Clone(ctx, "")
	if err != nil {
		return nil, processGitErrorf(err, "failed to clone shared repo with tag '%s'", request.GetTagName())
	}

	if err = sharedRepo.PushDeleteTag(ctx, base, request.GetTagName()); err != nil {
		return nil, processGitErrorf(err, "Failed to push the tag %s to remote", request.GetTagName())
	}

	return &rpc.UpdateRefResponse{}, nil
}
