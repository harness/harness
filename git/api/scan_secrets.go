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
	"fmt"

	"github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/report"
	"github.com/zricethezav/gitleaks/v8/sources"
)

// Finding contains gitleaks report.finding information.
type Finding struct {
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
	// unique identifier
	Fingerprint string `json:"fingerprint"`
}

// ScanSecrets detects secret leakage using gitleaks on the provided revision,
// or on the diff with baseRev if provided (baseRev is optional).
func (g *Git) ScanSecrets(
	repoPath string,
	baseRev string,
	rev string,
) ([]Finding, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	detector, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create a new gitleaks detector with default config: %w", err)
	}

	var logOpts string
	logOpts = fmt.Sprintf("--no-merges --first-parent %s", rev)

	if baseRev != "" {
		logOpts = fmt.Sprintf("--no-merges --first-parent %s..%s", baseRev, rev)
	}

	gitCmd, err := sources.NewGitLogCmd(repoPath, logOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new git log cmd with diff: %w", err)
	}

	res, err := detector.DetectGit(gitCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to detect git leaks on diff: %w", err)
	}

	return mapFindings(res), nil
}

func mapFindings(reports []report.Finding) []Finding {
	var findings []Finding
	for _, f := range reports {
		findings = append(findings, Finding{
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
