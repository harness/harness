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
	"fmt"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"

	"github.com/fatih/color"
)

var (
	colorScanHeaderFound = color.New(color.BgRed, color.FgHiWhite, color.Bold)
)

func printScanSecretsFindings(out *hook.Output, findings []api.Finding) {
	findingsCnt := len(findings)
	out.Messages = append(
		out.Messages,
		colorScanHeaderFound.Sprintf(
			" Detected leaked %s ",
			stringSecretOrSecrets(findingsCnt > 1),
		),
	)
	for _, finding := range findings {
		out.Messages = append(
			out.Messages,
			fmt.Sprintf("  Commit:   %s", finding.Commit),
			fmt.Sprintf("  File:     %s", finding.File),
		)
		if finding.StartLine == finding.EndLine {
			out.Messages = append(
				out.Messages,
				fmt.Sprintf("  Line:     %d", finding.StartLine),
			)
		} else {
			out.Messages = append(
				out.Messages,
				fmt.Sprintf("  Lines:    %d-%d", finding.StartLine, finding.EndLine),
			)
		}
		out.Messages = append(
			out.Messages,
			fmt.Sprintf("  Details:  %s", finding.Description),
			fmt.Sprintf("  Secret:   %s", finding.Match),
			fmt.Sprintf("  RuleID:   %s", finding.RuleID),
			fmt.Sprintf("  Author:   %s", finding.Author),
			fmt.Sprintf("  Date:     %s", finding.Date),
			"",
		)
	}
}

func stringSecretOrSecrets(plural bool) string {
	if plural {
		return "secrets"
	}
	return "secret"
}
