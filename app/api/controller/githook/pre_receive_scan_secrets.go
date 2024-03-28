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

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/logging"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

type scanSecretsResult struct {
	findings []api.Finding
}

func (r *scanSecretsResult) HasResults() bool {
	return len(r.findings) > 0
}

func (c *Controller) scanSecrets(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.Repository,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	// check if scanning is enabled on the repo
	scanningEnabled, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeySecretScanningEnabled,
		settings.DefaultSecretScanningEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to check settings whether secret scanning is enabled: %w", err)
	}
	if !scanningEnabled {
		return nil
	}

	// scan for secrets
	scanResult, err := scanSecretsInternal(
		ctx,
		rgit,
		repo,
		in,
	)
	if err != nil {
		return fmt.Errorf("failed to scan for git leaks: %w", err)
	}

	if !scanResult.HasResults() {
		return nil
	}

	// pretty print output
	printScanSecretsFindings(output, scanResult.findings)
	output.Messages = append(output.Messages, "", "")
	output.Error = ptr.String("Changes blocked by security scan results")

	return nil
}

func scanSecretsInternal(ctx context.Context,
	rgit RestrictedGIT,
	repo *types.Repository,
	in types.GithookPreReceiveInput,
) (scanSecretsResult, error) {
	var latestDfltCommitSHA string
	res := scanSecretsResult{}

	for _, refUpdate := range in.RefUpdates {
		ctx := logging.NewContext(ctx, loggingWithRefUpdate(refUpdate))
		log := log.Ctx(ctx)

		if refUpdate.New.String() == types.NilSHA {
			log.Debug().Msg("skip deleted reference")
			continue
		}

		// in case the branch was just created - fallback to compare against latest default branch.
		baseRev := refUpdate.Old.String() + "^{commit}"
		rev := refUpdate.New.String() + "^{commit}"
		//nolint:nestif
		if refUpdate.Old.String() == types.NilSHA {
			if latestDfltCommitSHA == "" {
				branchOut, err := rgit.GetBranch(ctx, &git.GetBranchParams{
					ReadParams: git.CreateReadParams(repo), // without any custom environment
					BranchName: repo.DefaultBranch,
				})
				if errors.IsNotFound(err) {
					return scanSecretsResult{}, nil
				}
				if err != nil {
					return scanSecretsResult{}, fmt.Errorf("failed to retrieve latest commit of default branch: %w", err)
				}
				latestDfltCommitSHA = branchOut.Branch.SHA.String()
			}
			baseRev = latestDfltCommitSHA

			log.Debug().Msgf("use latest dflt commit %s as comparison for new branch", latestDfltCommitSHA)
		}

		log.Debug().Msg("scan for secrets")

		scanSecretsOut, err := rgit.ScanSecrets(ctx, &git.ScanSecretsParams{
			ReadParams: git.ReadParams{
				RepoUID:             repo.GitUID,
				AlternateObjectDirs: in.Environment.AlternateObjectDirs,
			},
			BaseRev: baseRev,
			Rev:     rev,
		})
		if err != nil {
			return scanSecretsResult{}, fmt.Errorf("failed to detect secret leaks: %w", err)
		}

		if len(scanSecretsOut.Findings) == 0 {
			log.Debug().Msg("no new secrets found")
			continue
		}

		log.Debug().Msgf("found %d new secrets", len(scanSecretsOut.Findings))

		res.findings = append(res.findings, scanSecretsOut.Findings...)
	}

	return res, nil
}
