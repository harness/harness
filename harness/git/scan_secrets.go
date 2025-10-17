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
	"io"
	"os"
	"path"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/rs/zerolog/log"
	"github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/report"
	"github.com/zricethezav/gitleaks/v8/sources"
)

const (
	DefaultGitleaksIgnorePath = ".gitleaksignore"
)

type ScanSecretsParams struct {
	ReadParams

	BaseRev string // optional to scan for secrets on diff between baseRev and Rev
	Rev     string

	GitleaksIgnorePath string // optional, keep empty to skip using .gitleaksignore file.
}

type ScanSecretsOutput struct {
	Findings []ScanSecretsFinding
}

type ScanSecretsFinding struct {
	Description string   `json:"description"`
	StartLine   int64    `json:"start_line"`
	EndLine     int64    `json:"end_line"`
	StartColumn int64    `json:"start_column"`
	EndColumn   int64    `json:"end_column"`
	Match       string   `json:"match"`
	Secret      string   `json:"secret"`
	File        string   `json:"file"`
	SymlinkFile string   `json:"symlink_file"`
	Commit      string   `json:"commit"`
	Entropy     float64  `json:"entropy"`
	Author      string   `json:"author"`
	Email       string   `json:"email"`
	Date        string   `json:"date"`
	Message     string   `json:"message"`
	Tags        []string `json:"tags"`
	RuleID      string   `json:"rule_id"`
	// Fingerprint is the unique identifier of the secret that can be used in .gitleaksignore files.
	Fingerprint string `json:"fingerprint"`
}

func (s *Service) ScanSecrets(ctx context.Context, params *ScanSecretsParams) (*ScanSecretsOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	var findings []ScanSecretsFinding
	err := sharedrepo.Run(ctx, nil, s.sharedRepoRoot, repoPath, func(sharedRepo *sharedrepo.SharedRepo) error {
		fsGitleaksIgnorePath, err := s.setupGitleaksIgnoreInSharedRepo(
			ctx,
			sharedRepo,
			params.Rev,
			params.GitleaksIgnorePath,
		)
		if err != nil {
			return fmt.Errorf("failed to setup .gitleaksignore file in share repo: %w", err)
		}

		detector, err := detect.NewDetectorDefaultConfig()
		if err != nil {
			return fmt.Errorf("failed to create a new gitleaks detector with default config: %w", err)
		}
		if fsGitleaksIgnorePath != "" {
			if err := detector.AddGitleaksIgnore(fsGitleaksIgnorePath); err != nil {
				return fmt.Errorf("failed to load .gitleaksignore file from path %q: %w", fsGitleaksIgnorePath, err)
			}
		}

		// TODO: fix issue where secrets in second-parent commits are not detected
		logOpts := fmt.Sprintf("--no-merges --first-parent %s", params.Rev)
		if params.BaseRev != "" {
			logOpts = fmt.Sprintf("--no-merges --first-parent %s..%s", params.BaseRev, params.Rev)
		}

		gitCmd, err := sources.NewGitLogCmd(sharedRepo.Directory(), logOpts)
		if err != nil {
			return fmt.Errorf("failed to create a new git log cmd with diff: %w", err)
		}

		res, err := detector.DetectGit(gitCmd)
		if err != nil {
			return fmt.Errorf("failed to detect git leaks on diff: %w", err)
		}

		findings = mapFindings(res)

		return nil
	}, params.AlternateObjectDirs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaks on diff: %w", err)
	}

	return &ScanSecretsOutput{
		Findings: findings,
	}, nil
}

func (s *Service) setupGitleaksIgnoreInSharedRepo(
	ctx context.Context,
	sharedRepo *sharedrepo.SharedRepo,
	rev string,
	treePath string,
) (string, error) {
	if treePath == "" {
		log.Ctx(ctx).Debug().Msgf("no path to .gitleaksignore file provided, run without")
		return "", nil
	}

	// ensure file exists in tree for the provided revision, otherwise ignore and continue
	node, err := s.git.GetTreeNode(ctx, sharedRepo.Directory(), rev, treePath)
	if errors.IsNotFound(err) {
		log.Ctx(ctx).Debug().Msgf("no .gitleaksignore file found at %q, run without", treePath)
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get tree node for .gitleaksignore file at path %q: %w", treePath, err)
	}

	// if node isn't of type blob it won't contain any gitleaks related content - run without .gitleaksignore.
	// NOTE: We don't do any further checks for binary files or handle symlinks.
	if node.NodeType != api.TreeNodeTypeBlob {
		log.Ctx(ctx).Warn().Msgf(
			"tree node at path %q is of type %d instead of %d (blob), run without .gitleaksignore",
			treePath,
			node.NodeType,
			api.TreeNodeTypeBlob,
		)
		return "", nil
	}

	log.Ctx(ctx).Debug().Msgf(".gitleaksignore file found in tree at %q", treePath)

	// read blob data from bare repo
	blob, err := api.GetBlob(
		ctx,
		sharedRepo.Directory(),
		nil,
		node.SHA,
		0,
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get blob for .gitleaksignore file at path %q sha %q: %w", treePath, node.SHA, err)
	}
	defer func() {
		if err := blob.Content.Close(); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader for .gitleaksignore file")
		}
	}()

	// write file to root of .git folder of the shared repo for easy cleanup (it's a bare repo so otherwise no impact)
	filePath := path.Join(sharedRepo.Directory(), DefaultGitleaksIgnorePath)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to create tmp .gitleaksignore file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to close tmp .gitleaksignore file at %q", filePath)
		}
	}()

	_, err = io.Copy(f, blob.Content)
	if err != nil {
		return "", fmt.Errorf("failed to copy .gitleaksignore file from blob to %q: %w", filePath, err)
	}

	return filePath, nil
}

func mapFindings(reports []report.Finding) []ScanSecretsFinding {
	var findings []ScanSecretsFinding
	for _, f := range reports {
		findings = append(findings, ScanSecretsFinding{
			Description: f.Description,
			StartLine:   int64(f.StartLine),
			EndLine:     int64(f.EndLine),
			StartColumn: int64(f.StartColumn),
			EndColumn:   int64(f.EndColumn),
			Match:       f.Match,
			Secret:      f.Secret,
			File:        f.File,
			SymlinkFile: f.SymlinkFile,
			Commit:      f.Commit,
			Entropy:     float64(f.Entropy),
			Author:      f.Author,
			Email:       f.Email,
			Date:        f.Date,
			Message:     f.Message,
			Tags:        f.Tags,
			RuleID:      f.RuleID,
			Fingerprint: f.Fingerprint,
		})
	}
	return findings
}
