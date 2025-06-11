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
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/rs/zerolog/log"
)

var (
	listCommitTagsRefFields = []api.GitReferenceField{
		api.GitReferenceFieldRefName,
		api.GitReferenceFieldObjectType,
		api.GitReferenceFieldObjectName,
	}
	listCommitTagsObjectTypeFilter = []api.GitObjectType{
		api.GitObjectTypeCommit,
		api.GitObjectTypeTag,
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
	SHA         sha.SHA
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
	annotatedTagSHAs := make([]sha.SHA, 0, len(tags))
	commitSHAs := make([]sha.SHA, len(tags))

	for i, tag := range tags {
		// always set the commit sha (will be overwritten for annotated tags)
		commitSHAs[i] = tag.SHA

		if tag.IsAnnotated {
			annotatedTagSHAs = append(annotatedTagSHAs, tag.SHA)
		}
	}

	// populate annotation data for all annotated tags
	if len(annotatedTagSHAs) > 0 {
		var aTags []api.Tag
		aTags, err = s.git.GetAnnotatedTags(ctx, repoPath, annotatedTagSHAs)
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
			if aTags[ai].TargetType != api.GitObjectTypeCommit {
				ai++
				continue
			}

			// correct the commitSHA for the annotated tag (currently it is the tag sha, not the commit sha)
			commitSHAs[wi] = aTags[ai].TargetSHA

			tagger := mapSignature(aTags[ai].Tagger)

			// update tag information with annotation details
			// NOTE: we keep the name from the reference and ignore the annotated name (similar to github)
			tags[wi].Message = aTags[ai].Message
			tags[wi].Title = aTags[ai].Title
			tags[wi].Tagger = &tagger

			ai++
			wi++
		}

		// truncate slices based on what was removed
		tags = tags[:wi]
		commitSHAs = commitSHAs[:wi]
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if params.IncludeCommit {
		gitCommits, err := s.git.GetCommits(ctx, repoPath, commitSHAs)
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

	targetCommit, err := s.git.GetCommitFromRev(ctx, repoPath, params.Target)
	if errors.IsNotFound(err) {
		return nil, errors.NotFound("target '%s' doesn't exist", params.Target)
	}
	if err != nil {
		return nil, fmt.Errorf("CreateCommitTag: failed to get commit id for target '%s': %w", params.Target, err)
	}

	tagName := params.Name
	tagRef := api.GetReferenceFromTagName(tagName)
	var tag *api.Tag

	commitSHA, err := s.git.GetRef(ctx, repoPath, tagRef)
	// TODO: Change GetRef to use errors.NotFound and then remove types.IsNotFoundError(err) below.
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("CreateCommitTag: failed to verify tag existence: %w", err)
	}
	if err == nil && !commitSHA.IsEmpty() {
		return nil, errors.Conflict("tag '%s' already exists", tagName)
	}

	// create tag request

	tagger := params.Actor
	if params.Tagger != nil {
		tagger = *params.Tagger
	}
	taggerDate := time.Now().UTC()
	if params.TaggerDate != nil {
		taggerDate = *params.TaggerDate
	}

	createTagRequest := &api.CreateTagOptions{
		Message: params.Message,
		Tagger: api.Signature{
			Identity: api.Identity{
				Name:  tagger.Name,
				Email: tagger.Email,
			},
			When: taggerDate,
		},
	}

	// ref updater

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create ref updater to create the tag: %w", err)
	}

	// create the tag

	err = sharedrepo.Run(ctx, refUpdater, s.sharedRepoRoot, repoPath, func(r *sharedrepo.SharedRepo) error {
		if err := s.git.CreateTag(ctx, r.Directory(), tagName, targetCommit.SHA, createTagRequest); err != nil {
			return fmt.Errorf("failed to create tag '%s': %w", tagName, err)
		}

		tag, err = s.git.GetAnnotatedTag(ctx, r.Directory(), tagName)
		if err != nil {
			return fmt.Errorf("failed to read annotated tag after creation: %w", err)
		}

		ref := hook.ReferenceUpdate{
			Ref: tagRef,
			Old: sha.Nil,
			New: tag.Sha,
		}

		if err := refUpdater.Init(ctx, []hook.ReferenceUpdate{ref}); err != nil {
			return fmt.Errorf("failed to init ref updater: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("CreateCommitTag: failed to create tag in shared repository: %w", err)
	}

	// prepare response

	var commitTag *CommitTag
	if params.Message != "" {
		tag, err = s.git.GetAnnotatedTag(ctx, repoPath, params.Name)
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

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return fmt.Errorf("failed to create ref updater to delete the tag: %w", err)
	}

	tagRef := api.GetReferenceFromTagName(params.Name)

	err = refUpdater.DoOne(ctx, tagRef, sha.None, sha.Nil) // delete whatever is there
	if errors.IsNotFound(err) {
		return errors.NotFound("tag %q does not exist", params.Name)
	}
	if err != nil {
		return fmt.Errorf("failed to init ref updater: %w", err)
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

	opts := &api.WalkReferencesOptions{
		Patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixTag, params.Query),
		Sort:       mapListCommitTagsSortOption(params.Sort),
		Order:      mapToSortOrder(params.Order),
		Fields:     listCommitTagsRefFields,
		Instructor: instructor,
		// we do post-filtering, so we can't restrict the git output ...
		MaxWalkDistance: 0,
	}

	err = s.git.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to walk tag references: %w", err)
	}

	log.Ctx(ctx).Trace().Msgf("git api returned %d tags", len(tags))

	return tags, nil
}

func listCommitTagsWalkReferencesHandler(tags *[]CommitTag) api.WalkReferencesHandler {
	return func(e api.WalkReferencesEntry) error {
		fullRefName, ok := e[api.GitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[api.GitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}
		objectTypeRaw, ok := e[api.GitReferenceFieldObjectType]
		if !ok {
			return fmt.Errorf("entry missing object type")
		}

		tag := CommitTag{
			Name:        fullRefName[len(gitReferenceNamePrefixTag):],
			SHA:         sha.Must(objectSHA),
			IsAnnotated: objectTypeRaw == string(api.GitObjectTypeTag),
		}

		// TODO: refactor to not use slice pointers?
		*tags = append(*tags, tag)

		return nil
	}
}

func newInstructorWithObjectTypeFilter(filter []api.GitObjectType) api.WalkReferencesInstructor {
	return func(wre api.WalkReferencesEntry) (api.WalkInstruction, error) {
		v, ok := wre[api.GitReferenceFieldObjectType]
		if !ok {
			return api.WalkInstructionStop, fmt.Errorf("ref field for object type is missing")
		}

		// only handle if any of the filters match
		for _, field := range filter {
			if v == string(field) {
				return api.WalkInstructionHandle, nil
			}
		}

		// by default skip
		return api.WalkInstructionSkip, nil
	}
}
