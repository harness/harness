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
	"fmt"
	"math"
	"strings"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ReferenceService struct {
	rpc.UnimplementedReferenceServiceServer
	adapter   GitAdapter
	reposRoot string
	tmpDir    string
}

func NewReferenceService(adapter GitAdapter,
	reposRoot string, tmpDir string) (*ReferenceService, error) {
	return &ReferenceService{
		adapter:   adapter,
		reposRoot: reposRoot,
		tmpDir:    tmpDir,
	}, nil
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

// createReferenceWalkPatternsFromQuery returns a list of patterns that
// ensure only references matching the basePath and query are part of the walk.
func createReferenceWalkPatternsFromQuery(basePath string, query string) []string {
	if basePath == "" && query == "" {
		return []string{}
	}

	// ensure non-empty basepath ends with "/" for proper matching and concatenation.
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
		//nolint:goconst
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

// wrapInstructorWithOptionalPagination wraps the provided walkInstructor with pagination.
// If no paging is enabled, the original instructor is returned.
func wrapInstructorWithOptionalPagination(inner types.WalkReferencesInstructor,
	page int32, pageSize int32) (types.WalkReferencesInstructor, int32, error) {
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
	return func(e types.WalkReferencesEntry) (types.WalkInstruction, error) {
			// execute inner instructor
			inst, err := inner(e)
			if err != nil {
				return inst, err
			}

			// no pagination if element is filtered out
			if inst != types.WalkInstructionHandle {
				return inst, nil
			}

			// increase count iff element is part of filtered output
			c++

			// add pagination on filtered output
			switch {
			case c <= startAfter:
				return types.WalkInstructionSkip, nil
			case c > endAfter:
				return types.WalkInstructionStop, nil
			default:
				return types.WalkInstructionHandle, nil
			}
		},
		endAfter,
		nil
}

func (s ReferenceService) GetRef(ctx context.Context,
	request *rpc.GetRefRequest,
) (*rpc.GetRefResponse, error) {
	if request.Base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}
	repoPath := getFullPathForRepo(s.reposRoot, request.Base.GetRepoUid())

	refType := enum.RefFromRPC(request.GetRefType())
	if refType == enum.RefTypeUndefined {
		return nil, status.Error(codes.InvalidArgument, "invalid value of RefType argument")
	}
	reference, err := GetRefPath(request.GetRefName(), refType)
	if err != nil {
		return nil, err
	}

	sha, err := s.adapter.GetRef(ctx, repoPath, reference)
	if err != nil {
		return nil, err
	}

	return &rpc.GetRefResponse{Sha: sha}, nil
}

func (s ReferenceService) UpdateRef(ctx context.Context,
	request *rpc.UpdateRefRequest,
) (*rpc.UpdateRefResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	refType := enum.RefFromRPC(request.GetRefType())
	if refType == enum.RefTypeUndefined {
		return nil, status.Error(codes.InvalidArgument, "invalid value of RefType argument")
	}
	reference, err := GetRefPath(request.GetRefName(), refType)
	if err != nil {
		return nil, err
	}

	if ok, err := repoIsEmpty(ctx, repoPath); ok {
		return nil, ErrInvalidArgumentf("reference cannot be updated on empty repository", err)
	}

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

	pushOpts := types.PushOptions{
		Remote: repoPath,
		Env:    CreateEnvironmentForPush(ctx, base),
	}

	// handle deletion explicitly to avoid any unwanted side effects
	if request.GetNewValue() == "" {
		pushOpts.Branch = ":" + reference
	} else {
		pushOpts.Branch = request.GetNewValue() + ":" + reference
	}

	if request.GetOldValue() == "" {
		pushOpts.Force = true
	} else {
		pushOpts.ForceWithLease = reference + ":" + request.GetOldValue()
	}

	// TODO: why are we using gitea operations here?
	// TODO: our shared repo has so much duplication, that should be changed IMHO.
	err = gitea.Push(ctx, sharedRepo.tmpPath, pushOpts)
	if err != nil {
		return nil, processGitErrorf(err, "failed to push changes to original repo")
	}

	return &rpc.UpdateRefResponse{}, nil
}

func GetRefPath(refName string, refType enum.RefType) (string, error) {
	const (
		refPullReqPrefix      = "refs/pullreq/"
		refPullReqHeadSuffix  = "/head"
		refPullReqMergeSuffix = "/merge"
	)

	switch refType {
	case enum.RefTypeRaw:
		return refName, nil
	case enum.RefTypeBranch:
		return git.BranchPrefix + refName, nil
	case enum.RefTypeTag:
		return git.TagPrefix + refName, nil
	case enum.RefTypePullReqHead:
		return refPullReqPrefix + refName + refPullReqHeadSuffix, nil
	case enum.RefTypePullReqMerge:
		return refPullReqPrefix + refName + refPullReqMergeSuffix, nil
	case enum.RefTypeUndefined:
		fallthrough
	default:
		return "", ErrInvalidArgumentf("provided reference type '%s' is invalid", refType)
	}
}
