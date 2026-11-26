// Copyright 2026 Harness, Inc.
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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

var ctlNilSHA = sha.Must(strings.Repeat("0", 40))

const (
	ctlSHA1 = "1111111111111111111111111111111111111111"
	ctlSHA2 = "2222222222222222222222222222222222222222"
)

func TestCtlPrintScanSecretsFindings(t *testing.T) {
	t.Run("no findings", func(t *testing.T) {
		out := &hook.Output{}
		printScanSecretsFindings(out, nil, false, 1500*time.Millisecond)
		require.Len(t, out.Messages, 3)
		require.Contains(t, out.Messages[0], "No secrets found")
		require.Contains(t, out.Messages[0], "1.5s")
	})

	t.Run("findings", func(t *testing.T) {
		out := &hook.Output{}
		findings := []secretFinding{
			{
				ScanSecretsFinding: git.ScanSecretsFinding{
					RuleID: "aws-key", File: "a.go", StartLine: 3, EndLine: 3, Secret: "xxx",
					Commit: ctlSHA1, Description: "desc", Fingerprint: "fp",
				},
				Ref: "refs/heads/main",
			},
			{
				ScanSecretsFinding: git.ScanSecretsFinding{
					RuleID: "token", File: "b.go", StartLine: 3, EndLine: 7,
				},
				Ref: "refs/heads/dev",
			},
		}
		printScanSecretsFindings(out, findings, true, time.Second)

		joined := strings.Join(out.Messages, "\n")
		require.Contains(t, joined, "Push contains secrets:")
		require.Contains(t, joined, "aws-key in a.go:3 [refs/heads/main]")
		require.Contains(t, joined, "token in b.go:3-7 [refs/heads/dev]")
		require.Contains(t, joined, "2 secrets found")
	})

	t.Run("single finding without refs", func(t *testing.T) {
		out := &hook.Output{}
		printScanSecretsFindings(out, []secretFinding{{ScanSecretsFinding: git.ScanSecretsFinding{RuleID: "r"}}}, false, 0)
		joined := strings.Join(out.Messages, "\n")
		require.Contains(t, joined, "Push contains secret:")
		require.Contains(t, joined, "1 secret found")
	})
}

func TestCtlFMTDuration(t *testing.T) {
	require.Equal(t, "500ns", FMTDuration(500*time.Nanosecond))
	require.Equal(t, "1ms", FMTDuration(time.Millisecond))
	require.Equal(t, "123ms", FMTDuration(123400*time.Microsecond))
	require.Equal(t, "1.2s", FMTDuration(1234*time.Millisecond))
	require.Equal(t, "2m4s", FMTDuration(2*time.Minute+3600*time.Millisecond))
}

func TestCtlPrintOversizeFiles(t *testing.T) {
	// Nil inputs and empty results are no-ops.
	printOversizeFiles(nil, &git.FindOversizeFilesOutput{})
	out := &hook.Output{}
	printOversizeFiles(out, nil)
	printOversizeFiles(out, &git.FindOversizeFilesOutput{})
	require.Empty(t, out.Messages)

	findOut := &git.FindOversizeFilesOutput{
		TotalsPerLimit: map[int64]int64{100: 3, 50: 1, 10: 0},
		FileInfosPerLimit: map[int64][]git.FileInfo{
			100: {{SHA: sha.Must(ctlSHA1), Size: 200}, {SHA: sha.Must(ctlSHA2), Size: 150}},
			50:  {{SHA: sha.Must(ctlSHA1), Size: 70}},
		},
	}
	printOversizeFiles(out, findOut)

	joined := strings.Join(out.Messages, "\n")
	require.Contains(t, joined, "Push contains 3 files exceeding the size limit of 100B")
	require.Contains(t, joined, "Showing 2 of 3 files")
	require.Contains(t, joined, "Push contains 1 file exceeding the size limit of 50B (in addition to the 3 files above)")
	require.NotContains(t, joined, "limit of 10B")
	// The largest limit is printed first.
	require.Less(t, strings.Index(joined, "limit of 100B"), strings.Index(joined, "limit of 50B"))
}

func TestCtlPrintCommitterMismatchAndLFS(t *testing.T) {
	out := &hook.Output{}
	printCommitterMismatch(out, []git.CommitInfo{
		{SHA: sha.Must(ctlSHA1), Committer: "other@example.com"},
	}, "me@example.com", 1)
	joined := strings.Join(out.Messages, "\n")
	require.Contains(t, joined, "me@example.com")
	require.Contains(t, joined, "Committer: other@example.com")
	require.Contains(t, joined, "1 commit found not matching")

	out = &hook.Output{}
	printCommitterMismatch(out, nil, "me@example.com", 5)
	require.Contains(t, strings.Join(out.Messages, "\n"), "5 commits found")

	out = &hook.Output{}
	printLFSPointers(out, []git.LFSInfo{{ObjID: "abc", SHA: sha.Must(ctlSHA2)}}, 1)
	joined = strings.Join(out.Messages, "\n")
	require.Contains(t, joined, "Object ID: abc")
	require.Contains(t, joined, "1 LFS object missing")

	out = &hook.Output{}
	printLFSPointers(out, nil, 2)
	require.Contains(t, strings.Join(out.Messages, "\n"), "2 LFS objects missing")
}

func TestCtlQuotaPrinting(t *testing.T) {
	// Non quota errors are not handled.
	out := &hook.Output{}
	require.False(t, printRepoQuotaLimit(out, errors.New("boom")))
	require.False(t, printTotalStorageQuotaLimit(out, errors.New("boom")))
	require.False(t, printRepoQuotaLimit(out, nil))
	require.Empty(t, out.Messages)

	const mib = int64(1 << 20)

	for _, level := range []limiter.RepoQuotaStorageLevel{
		limiter.RepoQuotaLevelSoft, limiter.RepoQuotaLevelHard, limiter.RepoQuotaLevelOverLimit,
	} {
		out = &hook.Output{}
		require.True(t, printRepoQuotaLimit(out, &limiter.RepoQuotaStorageError{
			Level: level, Size: 12 * mib, SoftLimit: 5 * mib, HardLimit: 10 * mib,
		}))
		require.NotEmpty(t, out.Messages)
		require.Contains(t, strings.Join(out.Messages, "\n"), "Repository storage")
		if level == limiter.RepoQuotaLevelOverLimit {
			require.NotNil(t, out.Error)
			require.Contains(t, *out.Error, "Repository storage limit exceeded")
		} else {
			require.Nil(t, out.Error)
		}
	}

	for _, level := range []limiter.TotalQuotaStorageLevel{
		limiter.TotalQuotaLevelWarn, limiter.TotalQuotaLevelCritical, limiter.TotalQuotaLevelOverLimit,
	} {
		out = &hook.Output{}
		require.True(t, printTotalStorageQuotaLimit(out, &limiter.TotalQuotaStorageError{
			Level: level, Size: 12 * mib, Limit: 10 * mib, WarnPercent: 50, CriticalPercent: 80,
		}))
		require.Contains(t, strings.Join(out.Messages, "\n"), "Total storage")
	}

	// A quota error wrapped in another error is recognized too.
	out = &hook.Output{}
	wrapped := errors.Join(errors.New("ctx"), &limiter.RepoQuotaStorageError{Level: limiter.RepoQuotaLevelSoft})
	require.True(t, printRepoQuotaLimit(out, wrapped))
}

func TestCtlQuotaHelpers(t *testing.T) {
	require.EqualValues(t, 0, percentOfLimit(0, 50))
	require.EqualValues(t, 0, percentOfLimit(100, 0))
	require.EqualValues(t, 50, percentOfLimit(100, 50))

	require.Equal(t, "0 B", formatQuotaBytes(-5))
	require.Equal(t, "1.0 KiB", formatQuotaBytes(1024))

	require.Empty(t, quotaProgressBar(10, 1, 2, 0))
	require.NotEmpty(t, quotaProgressBar(5, 5, 8, 10))
	require.NotEmpty(t, quotaProgressBar(50, 5, 8, 10), "usage beyond the limit still renders")
	require.NotEmpty(t, quotaProgressBar(0, 0, 0, 10))

	require.Equal(t, quotaStatusOK, repoQuotaStatus(""))
	require.Equal(t, quotaStatusWarning, repoQuotaStatus(limiter.RepoQuotaLevelSoft))
	require.Equal(t, quotaStatusOverLimit, repoQuotaStatus(limiter.RepoQuotaLevelHard))
	require.Equal(t, quotaStatusOK, totalQuotaStatus(""))
	require.Equal(t, quotaStatusWarning, totalQuotaStatus(limiter.TotalQuotaLevelWarn))
	require.Equal(t, quotaStatusCritical, totalQuotaStatus(limiter.TotalQuotaLevelCritical))
	require.Equal(t, quotaStatusOverLimit, totalQuotaStatus(limiter.TotalQuotaLevelOverLimit))

	require.Equal(t, "file", singularOrPlural("file", false))
	require.Equal(t, "files", singularOrPlural("file", true))
}

func TestCtlRefGrouping(t *testing.T) {
	old, updated := sha.Must(ctlSHA1), sha.Must(ctlSHA2)
	updates := []hook.ReferenceUpdate{
		{Ref: "refs/heads/created", Old: ctlNilSHA, New: updated},
		{Ref: "refs/heads/deleted", Old: old, New: ctlNilSHA},
		{Ref: "refs/heads/updated", Old: old, New: updated},
		{Ref: "refs/heads/forced", Old: old, New: updated},
		{Ref: "refs/tags/v1", Old: old, New: updated},
		{Ref: "refs/pullreq/1/head", Old: old, New: updated},
	}
	forced := []bool{false, false, false, true, true, true}

	refs := groupRefsByAction(updates, forced)
	require.Equal(t, []string{"created"}, refs.branches.created)
	require.Equal(t, []string{"deleted"}, refs.branches.deleted)
	require.Equal(t, []string{"updated"}, refs.branches.updated)
	require.Equal(t, []string{"forced"}, refs.branches.forced)
	require.Equal(t, []string{"v1"}, refs.tags.forced)
	// Other refs are never reported as forced.
	require.Equal(t, []string{"refs/pullreq/1/head"}, refs.other.updated)

	require.False(t, refs.hasOnlyDeletedBranches())
	only := groupRefsByAction(updates[1:2], []bool{false})
	require.True(t, only.hasOnlyDeletedBranches())

	require.True(t, isBranch("refs/heads/x"))
	require.False(t, isBranch("refs/tags/x"))
	require.True(t, isTag("refs/tags/x"))
	require.False(t, isTag("refs/heads/x"))

	require.NotNil(t, loggingWithRefUpdate(updates[0]))
}

func TestCtlRepoSettings(t *testing.T) {
	limit := int64(100)
	zero := int64(0)

	require.EqualValues(t, 100, repoSettings{FileSizeLimit: &limit}.fileSizeLimit(true))
	require.EqualValues(t, 100, repoSettings{FileSizeLimit: &limit}.fileSizeLimit(false))
	require.EqualValues(t, 0, repoSettings{}.fileSizeLimit(true))
	require.EqualValues(t, settings.DefaultFileSizeLimit, repoSettings{}.fileSizeLimit(false))
	require.EqualValues(t, settings.DefaultFileSizeLimit, repoSettings{FileSizeLimit: &zero}.fileSizeLimit(false))

	require.False(t, repoSettings{}.enabled())
	require.True(t, repoSettings{SecretScanningEnabled: true}.enabled())
	require.True(t, repoSettings{FileSizeLimit: &limit}.enabled())
	require.True(t, repoSettings{PrincipalCommitterMatch: true}.enabled())
	require.True(t, repoSettings{GitLFSEnabled: true}.enabled())
}

type ctlRestrictedGit struct {
	RestrictedGIT
	ancestor bool
	err      error
}

func (g ctlRestrictedGit) IsAncestor(context.Context, git.IsAncestorParams) (git.IsAncestorOutput, error) {
	return git.IsAncestorOutput{Ancestor: g.ancestor}, g.err
}

func TestCtlIsForcePush(t *testing.T) {
	ctx := context.Background()
	old, updated := sha.Must(ctlSHA1), sha.Must(ctlSHA2)

	tests := []struct {
		name   string
		update hook.ReferenceUpdate
		git    ctlRestrictedGit
		want   bool
	}{
		{"create", hook.ReferenceUpdate{Ref: "refs/heads/a", Old: ctlNilSHA, New: updated}, ctlRestrictedGit{}, false},
		{"delete", hook.ReferenceUpdate{Ref: "refs/heads/a", Old: old, New: ctlNilSHA}, ctlRestrictedGit{}, false},
		{"tag update", hook.ReferenceUpdate{Ref: "refs/tags/a", Old: old, New: updated}, ctlRestrictedGit{}, true},
		{"fast forward", hook.ReferenceUpdate{Ref: "refs/heads/a", Old: old, New: updated},
			ctlRestrictedGit{ancestor: true}, false},
		{"rewrite", hook.ReferenceUpdate{Ref: "refs/heads/a", Old: old, New: updated},
			ctlRestrictedGit{ancestor: false}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isForcePush(ctx, tt.git, "uid", nil, tt.update)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	errGit := errors.New("git failure")
	_, err := isForcePush(ctx, ctlRestrictedGit{err: errGit}, "uid", nil,
		hook.ReferenceUpdate{Ref: "refs/heads/a", Old: old, New: updated})
	require.ErrorIs(t, err, errGit)
}

func TestCtlLogOutputFor(t *testing.T) {
	ctx := context.Background()
	msgs := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		msgs = append(msgs, "line", "")
	}
	errMsg := "failed"

	// Only verifies that logging does not panic for all shapes of output.
	logOutputFor(ctx, "pre-receive", hook.Output{})
	logOutputFor(ctx, "pre-receive", hook.Output{Error: &errMsg})
	logOutputFor(ctx, "pre-receive", hook.Output{Messages: []string{"a", "", "b"}})
	logOutputFor(ctx, "pre-receive", hook.Output{Messages: msgs, Error: &errMsg})
}
