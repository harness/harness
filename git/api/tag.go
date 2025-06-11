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

package api

import (
	"context"
	"io"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"
)

type Tag struct {
	Sha        sha.SHA
	Name       string
	TargetSHA  sha.SHA
	TargetType GitObjectType
	Title      string
	Message    string
	Tagger     Signature
	SignedData *SignedData
}

type CreateTagOptions struct {
	// Message is the optional message the tag will be created with - if the message is empty
	// the tag will be lightweight, otherwise it'll be annotated.
	Message string

	// Tagger is the information used in case the tag is annotated (Message is provided).
	Tagger Signature
}

// TagPrefix tags prefix path on the repository.
const TagPrefix = "refs/tags/"

// GetAnnotatedTag returns the tag for a specific tag sha.
func (g *Git) GetAnnotatedTag(
	ctx context.Context,
	repoPath string,
	rev string,
) (*Tag, error) {
	return CatFileAnnotatedTag(ctx, repoPath, nil, rev)
}

// GetAnnotatedTags returns the tags for a specific list of tag sha.
func (g *Git) GetAnnotatedTags(
	ctx context.Context,
	repoPath string,
	tagSHAs []sha.SHA,
) ([]Tag, error) {
	return CatFileAnnotatedTagFromSHAs(ctx, repoPath, nil, tagSHAs)
}

// CreateTag creates the tag pointing at the provided SHA (could be any type, e.g. commit, tag, blob, ...)
func (g *Git) CreateTag(
	ctx context.Context,
	repoPath string,
	name string,
	targetSHA sha.SHA,
	opts *CreateTagOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	cmd := command.New("tag")
	if opts != nil && opts.Message != "" {
		cmd.Add(command.WithFlag("-m", opts.Message))
		cmd.Add(
			command.WithCommitterAndDate(
				opts.Tagger.Identity.Name,
				opts.Tagger.Identity.Email,
				opts.Tagger.When,
			),
		)
	}

	cmd.Add(command.WithArg(name, targetSHA.String()))
	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		return processGitErrorf(err, "Service failed to create a tag")
	}
	return nil
}

func (g *Git) GetTagCount(
	ctx context.Context,
	repoPath string,
) (int, error) {
	if repoPath == "" {
		return 0, ErrRepositoryPathEmpty
	}

	pipeOut, pipeIn := io.Pipe()
	defer pipeOut.Close()

	cmd := command.New("tag")

	go func() {
		err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(pipeIn))
		if err != nil {
			_ = pipeIn.CloseWithError(
				processGitErrorf(err, "failed to trigger tag command"),
			)
			return
		}
		_ = pipeIn.Close()
	}()

	return countLines(pipeOut)
}
