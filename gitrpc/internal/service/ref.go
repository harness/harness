// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

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

	refType := enum.RefFromRPC(request.GetRefType())
	if refType == enum.RefTypeUndefined {
		return nil, status.Error(codes.InvalidArgument, "invalid value of RefType argument")
	}

	repoPath := getFullPathForRepo(s.reposRoot, request.Base.GetRepoUid())

	sha, err := s.adapter.GetRef(ctx, repoPath, request.RefName, refType)
	if err != nil {
		return nil, err
	}

	return &rpc.GetRefResponse{Sha: sha}, nil
}

func (s ReferenceService) UpdateRef(ctx context.Context,
	request *rpc.UpdateRefRequest,
) (*rpc.UpdateRefResponse, error) {
	if request.Base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	refType := enum.RefFromRPC(request.GetRefType())
	if refType == enum.RefTypeUndefined {
		return nil, status.Error(codes.InvalidArgument, "invalid value of RefType argument")
	}

	repoPath := getFullPathForRepo(s.reposRoot, request.Base.GetRepoUid())

	err := s.adapter.UpdateRef(ctx, repoPath, request.RefName, refType, request.NewValue, request.OldValue)
	if err != nil {
		return nil, err
	}

	return &rpc.UpdateRefResponse{}, nil
}
