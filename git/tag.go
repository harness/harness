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
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/adapter"
	"github.com/harness/gitness/git/types"

	"github.com/rs/zerolog/log"
)

var (
	listCommitTagsRefFields = []types.GitReferenceField{
		types.GitReferenceFieldRefName,
		types.GitReferenceFieldObjectType,
		types.GitReferenceFieldObjectName,
	}
	listCommitTagsObjectTypeFilter = []types.GitObjectType{
		types.GitObjectTypeCommit,
		types.GitObjectTypeTag,
	}
)

type TagSortOption int

const (
	TagSortOptionDefault TagSortOption = iota
	TagSortOptionName
	TagSortOptionDate
)

type ListCommitTagsParams struct {
	ReadParams
	IncludeCommit bool
	Query         string
	Sort          TagSortOption
	Order         SortOrder
	Page          int32
	PageSize      int32
}

type ListCommitTagsOutput struct {
	Tags []CommitTag
}

type CommitTag struct {
	Name        string
	SHA         string
	IsAnnotated bool
	Title       string
	Message     string
	Tagger      *Signature
	Commit      *Commit
}

type CreateCommitTagParams struct {
	WriteParams
	Name string

	// Target is the commit (or points to the commit) the new tag will be pointing to.
	Target string

	// Message is the optional message the tag will be created with - if the message is empty
	// the tag will be lightweight, otherwise it'll be annotated
	Message string

	// Tagger overwrites the git author used in case the tag is annotated
	// (optional, default: actor)
	Tagger *Identity
	// TaggerDate overwrites the git author date used in case the tag is annotated
	// (optional, default: current time on server)
	TaggerDate *time.Time
}

func (p *CreateCommitTagParams) Validate() error {
	if p == nil {
		return ErrNoParamsProvided
	}

	if p.Name == "" {
		return errors.New("tag name cannot be empty")
	}
	if p.Target == "" {
		return errors.New("target cannot be empty")
	}

	return nil
}

type CreateCommitTagOutput struct {
	CommitTag
}

type DeleteTagParams struct {
	WriteParams
	Name string
}

func (p DeleteTagParams) Validate() error {
	if p.Name == "" {
		return errors.New("tag name cannot be empty")
	}
	return nil
}

//nolint:gocognit
func (s *Service) ListCommitTags(
	ctx context.Context,
	params *ListCommitTagsParams,
) (*ListCommitTagsOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	// get all required information from git references
	tags, err := s.listCommitTagsLoadReferenceData(ctx, repoPath, params)
	if err != nil {
		return nil, fmt.Errorf("ListCommitTags: failed to get git references: %w", err)
	}

	// get all tag and commit SHAs
	annotatedTagSHAs := make([]string, 0, len(tags))
	commitSHAs := make([]string, len(tags))

	for i, tag := range tags {
		// always set the commit sha (will be overwritten for annotated tags)
		commitSHAs[i] = tag.SHA

		if tag.IsAnnotated {
			annotatedTagSHAs = append(annotatedTagSHAs, tag.SHA)
		}
	}

	// populate annotation data for all annotated tags
	if len(annotatedTagSHAs) > 0 {
		var aTags []types.Tag
		aTags, err = s.adapter.GetAnnotatedTags(ctx, repoPath, annotatedTagSHAs)
		if err != nil {
			return nil, fmt.Errorf("ListCommitTags: failed to get annotated tag: %w", err)
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
			tagger, err := mapSignature(&aTags[ai].Tagger)
			if err != nil {
				return nil, fmt.Errorf("signature mapping error: %w", err)
			}
			tags[wi].Tagger = tagger

			ai++
			wi++
		}

		// truncate slices based on what was removed
		tags = tags[:wi]
		commitSHAs = commitSHAs[:wi]
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if params.IncludeCommit {
		gitCommits, err := s.adapter.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return nil, fmt.Errorf("ListCommitTags: failed to get commits: %w", err)
		}

		for i := range gitCommits {
			c, err := mapCommit(&gitCommits[i])
			if err != nil {
				return nil, fmt.Errorf("commit mapping error: %w", err)
			}
			tags[i].Commit = c
		}
	}

	return &ListCommitTagsOutput{
		Tags: tags,
	}, nil
}

//nolint:gocognit
func (s *Service) CreateCommitTag(ctx context.Context, params *CreateCommitTagParams) (*CreateCommitTagOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	targetCommit, err := s.adapter.GetCommit(ctx, repoPath, params.Target)
	if errors.IsNotFound(err) {
		return nil, errors.NotFound("target '%s' doesn't exist", params.Target)
	}
	if err != nil {
		return nil, fmt.Errorf("CreateCommitTag: failed to get commit id for target '%s': %w", params.Target, err)
	}

	tagName := params.Name
	tagRef := adapter.GetReferenceFromTagName(tagName)
	var tag *types.Tag

	sha, err := s.adapter.GetRef(ctx, repoPath, tagRef)
	// TODO: Change GetRef to use errors.NotFound and then remove types.IsNotFoundError(err) below.
	if err != nil && !types.IsNotFoundError(err) && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("CreateCommitTag: failed to verify tag existence: %w", err)
	}
	if err == nil && sha != "" {
		return nil, errors.Conflict("tag '%s' already exists", tagName)
	}

	err = func() error {
		// Create a directory for the temporary shared repository.
		sharedRepo, err := s.adapter.SharedRepository(s.tmpDir, params.RepoUID, repoPath)
		if err != nil {
			return fmt.Errorf("failed to create new shared repo: %w", err)
		}
		defer sharedRepo.Close(ctx)

		// Create bare repository with alternates pointing to the original repository.
		err = sharedRepo.InitAsShared(ctx)
		if err != nil {
			return fmt.Errorf("failed to create temp repo with alternates: %w", err)
		}

		tagger := params.Actor
		if params.Tagger != nil {
			tagger = *params.Tagger
		}
		taggerDate := time.Now().UTC()
		if params.TaggerDate != nil {
			taggerDate = *params.TaggerDate
		}

		createTagRequest := &types.CreateTagOptions{
			Message: params.Message,
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
			sharedRepo.Path(),
			tagName,
			targetCommit.SHA,
			createTagRequest)
		if err != nil {
			return fmt.Errorf("failed to create tag '%s': %w", tagName, err)
		}

		tag, err = s.adapter.GetAnnotatedTag(ctx, sharedRepo.Path(), tagName)
		if err != nil {
			return fmt.Errorf("failed to read annotated tag after creation: %w", err)
		}

		err = sharedRepo.MoveObjects(ctx)
		if err != nil {
			return fmt.Errorf("failed to move git objects: %w", err)
		}

		return nil
	}()
	if err != nil {
		return nil, fmt.Errorf("CreateCommitTag: failed to create tag in shared repository: %w", err)
	}

	err = s.adapter.UpdateRef(
		ctx,
		params.EnvVars,
		repoPath,
		tagRef,
		types.NilSHA,
		tag.Sha,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag reference: %w", err)
	}

	var commitTag *CommitTag
	if params.Message != "" {
		tag, err = s.adapter.GetAnnotatedTag(ctx, repoPath, params.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to read annotated tag after creation: %w", err)
		}
		commitTag = mapAnnotatedTag(tag)
	} else {
		commitTag = &CommitTag{
			Name:        params.Name,
			IsAnnotated: false,
			SHA:         targetCommit.SHA,
		}
	}

	c, err := mapCommit(targetCommit)
	if err != nil {
		return nil, err
	}
	commitTag.Commit = c

	return &CreateCommitTagOutput{CommitTag: *commitTag}, nil
}

func (s *Service) DeleteTag(ctx context.Context, params *DeleteTagParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	tagRef := adapter.GetReferenceFromTagName(params.Name)

	err := s.adapter.UpdateRef(
		ctx,
		params.EnvVars,
		repoPath,
		tagRef,
		"", // delete whatever is there
		types.NilSHA,
	)
	if types.IsNotFoundError(err) {
		return errors.NotFound("tag %q does not exist", params.Name)
	}
	if err != nil {
		return fmt.Errorf("failed to delete tag reference: %w", err)
	}

	return nil
}

func (s *Service) listCommitTagsLoadReferenceData(
	ctx context.Context,
	repoPath string,
	params *ListCommitTagsParams,
) ([]CommitTag, error) {
	// TODO: can we be smarter with slice allocation
	tags := make([]CommitTag, 0, 16)
	handler := listCommitTagsWalkReferencesHandler(&tags)
	instructor, _, err := wrapInstructorWithOptionalPagination(
		newInstructorWithObjectTypeFilter(listCommitTagsObjectTypeFilter),
		params.Page,
		params.PageSize,
	)
	if err != nil {
		return nil, errors.InvalidArgument("invalid pagination details: %v", err)
	}

	opts := &types.WalkReferencesOptions{
		Patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixTag, params.Query),
		Sort:       mapListCommitTagsSortOption(params.Sort),
		Order:      mapToSortOrder(params.Order),
		Fields:     listCommitTagsRefFields,
		Instructor: instructor,
		// we do post-filtering, so we can't restrict the git output ...
		MaxWalkDistance: 0,
	}

	err = s.adapter.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to walk tag references: %w", err)
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d tags", len(tags))

	return tags, nil
}

func listCommitTagsWalkReferencesHandler(tags *[]CommitTag) types.WalkReferencesHandler {
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

		tag := CommitTag{
			Name:        fullRefName[len(gitReferenceNamePrefixTag):],
			SHA:         objectSHA,
			IsAnnotated: objectTypeRaw == string(types.GitObjectTypeTag),
		}

		// TODO: refactor to not use slice pointers?
		*tags = append(*tags, tag)

		return nil
	}
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
