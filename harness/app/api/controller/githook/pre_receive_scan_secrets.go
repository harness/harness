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
	"time"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/logging"
	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

type secretFinding struct {
	git.ScanSecretsFinding
	Ref string
}

func (c *Controller) scanSecrets(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	scanningEnabled bool,
	violationsInput *protection.PushViolationsInput,
	in types.GithookPreReceiveInput,
	output *hook.Output,
) error {
	if !scanningEnabled {
		var err error
		scanningEnabled, err = settings.RepoGet(
			ctx,
			c.settings,
			repo.ID,
			settings.KeySecretScanningEnabled,
			settings.DefaultSecretScanningEnabled,
		)
		if err != nil {
			return fmt.Errorf("failed to check settings whether secret scanning is enabled: %w", err)
		}
	}

	if !scanningEnabled {
		return nil
	}

	// scan for secrets
	startTime := time.Now()
	findings, err := scanSecretsInternal(
		ctx,
		rgit,
		repo,
		in,
	)
	if err != nil {
		return fmt.Errorf("failed to scan for git leaks: %w", err)
	}

	// always print result (handles both no results and results found)
	printScanSecretsFindings(output, findings, len(in.RefUpdates) > 1, time.Since(startTime))

	// this will be removed when secret scanning check will be moved to push protection
	if len(findings) > 0 && violationsInput == nil {
		errMsg := fmt.Sprintf("Found %d secret(s) in your code. Push rejected.", len(findings))
		output.Error = ptr.String(errMsg)
	}

	if violationsInput != nil {
		violationsInput.SecretScanningEnabled = scanningEnabled
		violationsInput.FoundSecretCount = len(findings)
	}

	return nil
}

func scanSecretsInternal(ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	in types.GithookPreReceiveInput,
) ([]secretFinding, error) {
	var baseRevFallBack *string
	findings := []secretFinding{}

	for _, refUpdate := range in.RefUpdates {
		ctx := logging.NewContext(ctx, loggingWithRefUpdate(refUpdate))
		log := log.Ctx(ctx)

		if refUpdate.New.IsNil() {
			log.Debug().Msg("skip deleted reference")
			continue
		}

		// in case the branch was just created - fallback to compare against latest default branch.
		baseRev := refUpdate.Old.String() + "^{commit}" //nolint:goconst
		rev := refUpdate.New.String() + "^{commit}"     //nolint:goconst
		//nolint:nestif
		if refUpdate.Old.IsNil() {
			if baseRevFallBack == nil {
				fallbackSHA, fallbackAvailable, err := GetBaseSHAForScanningChanges(
					ctx,
					rgit,
					repo,
					in.Environment,
					in.RefUpdates,
					refUpdate,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to get fallback sha: %w", err)
				}

				if fallbackAvailable {
					log.Debug().Msgf("found fallback sha %q", fallbackSHA)
					baseRevFallBack = ptr.String(fallbackSHA.String())
				} else {
					log.Debug().Msg("no fallback sha available, do full scan instead")
					baseRevFallBack = ptr.String("")
				}
			}

			log.Debug().Msgf("new reference, use rev %q as base for secret scanning", *baseRevFallBack)

			baseRev = *baseRevFallBack
		}

		log.Debug().Msg("scan for secrets")

		scanSecretsOut, err := rgit.ScanSecrets(ctx, &git.ScanSecretsParams{
			ReadParams: git.ReadParams{
				RepoUID:             repo.GitUID,
				AlternateObjectDirs: in.Environment.AlternateObjectDirs,
			},
			BaseRev:            baseRev,
			Rev:                rev,
			GitleaksIgnorePath: git.DefaultGitleaksIgnorePath,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to detect secret leaks: %w", err)
		}

		if len(scanSecretsOut.Findings) == 0 {
			log.Debug().Msg("no new secrets found")
			continue
		}

		log.Debug().Msgf("found %d new secrets", len(scanSecretsOut.Findings))

		for _, finding := range scanSecretsOut.Findings {
			findings = append(findings, secretFinding{
				ScanSecretsFinding: finding,
				Ref:                refUpdate.Ref,
			})
		}
	}

	if len(findings) > 0 {
		log.Ctx(ctx).Debug().Msgf("found total of %d new secrets", len(findings))
	}

	return findings, nil
}
