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
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

const gc5LFSPointerFmt = "version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize 12345\n"

func gc5LFSPointer() (content, oid string) {
	oid = strings.Repeat("ab", 32)
	return strings.Replace(gc5LFSPointerFmt, "%s", oid, 1), oid
}

// gc5Quarantine writes git objects into a separate object directory, like receive-pack does for a push in flight.
type gc5Quarantine struct {
	dir  string
	f    *gc5Fixture
	objs string
}

func gc5NewQuarantine(t *testing.T, f *gc5Fixture) *gc5Quarantine {
	t.Helper()

	return &gc5Quarantine{dir: t.TempDir(), f: f, objs: filepath.Join(f.path, "objects")}
}

func (q *gc5Quarantine) run(ctx context.Context, t *testing.T, stdin string, env []string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = q.f.path
	cmd.Env = append(cmd.Environ(),
		"GIT_OBJECT_DIRECTORY="+q.dir,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES="+q.objs,
	)
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	require.NoErrorf(t, cmd.Run(), "git %v: %s", args, stderr.String())

	return strings.TrimSpace(stdout.String())
}

func (q *gc5Quarantine) blob(ctx context.Context, t *testing.T, content string) string {
	t.Helper()

	return q.run(ctx, t, content, nil, "hash-object", "-w", "--stdin")
}

// commit creates a commit on top of the main branch with the given blob as file.txt.
func (q *gc5Quarantine) commit(ctx context.Context, t *testing.T, blobSHA, committerEmail string) string {
	t.Helper()

	tree := q.run(ctx, t, "100644 blob "+blobSHA+"\tfile.txt\n", nil, "mktree")

	return q.run(ctx, t, "", []string{
		"GIT_AUTHOR_NAME=Gc5", "GIT_AUTHOR_EMAIL=" + committerEmail,
		"GIT_COMMITTER_NAME=Gc5", "GIT_COMMITTER_EMAIL=" + committerEmail,
	}, "commit-tree", tree, "-m", "quarantined")
}

func TestGc5ProcessPreReceiveObjectsNoParams(t *testing.T) {
	svc := gc5NewService(t)

	out, err := svc.ProcessPreReceiveObjects(t.Context(), ProcessPreReceiveObjectsParams{})
	require.NoError(t, err)
	require.Nil(t, out.FindOversizeFilesOutput)
	require.Nil(t, out.FindCommitterMismatchOutput)
	require.Nil(t, out.FindLFSPointersOutput)
}

func TestGc5ProcessPreReceiveObjects(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()
	q := gc5NewQuarantine(t, f)

	lfsContent, lfsOID := gc5LFSPointer()
	small := q.blob(ctx, t, "x\n")
	lfs := q.blob(ctx, t, lfsContent)
	big := q.blob(ctx, t, strings.Repeat("0123456789", 40)) // 400 bytes
	commit := q.commit(ctx, t, small, "someone-else@example.com")

	read := ReadParams{RepoUID: f.uid, AlternateObjectDirs: []string{q.dir}}

	out, err := f.svc.ProcessPreReceiveObjects(ctx, ProcessPreReceiveObjectsParams{
		ReadParams:                  read,
		FindOversizeFilesParams:     &FindOversizeFilesParams{SizeLimits: []int64{40, 200}},
		FindCommitterMismatchParams: &FindCommitterMismatchParams{PrincipalEmail: "gc5-actor@example.com"},
		FindLFSPointersParams:       &FindLFSPointersParams{},
	})
	require.NoError(t, err)

	// oversize: lfs pointer (~130 bytes) is above 40 only; the big blob is above 200.
	require.NotNil(t, out.FindOversizeFilesOutput)
	require.Equal(t, int64(2), out.FindOversizeFilesOutput.Total)
	require.Len(t, out.FindOversizeFilesOutput.FileInfosPerLimit[200], 1)
	require.Equal(t, big, out.FindOversizeFilesOutput.FileInfosPerLimit[200][0].SHA.String())
	require.Len(t, out.FindOversizeFilesOutput.FileInfosPerLimit[40], 1)
	require.Equal(t, lfs, out.FindOversizeFilesOutput.FileInfosPerLimit[40][0].SHA.String())
	require.Equal(t, int64(1), out.FindOversizeFilesOutput.AccumulatedTotal(200))
	require.Equal(t, int64(2), out.FindOversizeFilesOutput.AccumulatedTotal(40))

	// committer mismatch.
	require.NotNil(t, out.FindCommitterMismatchOutput)
	require.Equal(t, int64(1), out.FindCommitterMismatchOutput.Total)
	require.Equal(t, commit, out.FindCommitterMismatchOutput.CommitInfos[0].SHA.String())
	require.Equal(t, "someone-else@example.com", out.FindCommitterMismatchOutput.CommitInfos[0].Committer)

	// lfs pointers.
	require.NotNil(t, out.FindLFSPointersOutput)
	require.Equal(t, int64(1), out.FindLFSPointersOutput.Total)
	require.Equal(t, lfsOID, out.FindLFSPointersOutput.LFSInfos[0].ObjID)
	require.Equal(t, lfs, out.FindLFSPointersOutput.LFSInfos[0].SHA.String())

	// the same committer email (case-insensitively) is no mismatch.
	out, err = f.svc.ProcessPreReceiveObjects(ctx, ProcessPreReceiveObjectsParams{
		ReadParams:                  read,
		FindCommitterMismatchParams: &FindCommitterMismatchParams{PrincipalEmail: "SOMEONE-ELSE@example.com"},
	})
	require.NoError(t, err)
	require.Zero(t, out.FindCommitterMismatchOutput.Total)
}

func TestGc5ProcessPreReceiveObjectsNoCandidates(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()
	q := gc5NewQuarantine(t, f)

	q.blob(ctx, t, strings.Repeat("0123456789", 40))

	out, err := f.svc.ProcessPreReceiveObjects(ctx, ProcessPreReceiveObjectsParams{
		ReadParams:            ReadParams{RepoUID: f.uid, AlternateObjectDirs: []string{q.dir}},
		FindLFSPointersParams: &FindLFSPointersParams{},
	})
	require.NoError(t, err)
	require.Empty(t, out.FindLFSPointersOutput.LFSInfos)
	require.Zero(t, out.FindLFSPointersOutput.Total)
}

func TestGc5ProcessPreReceiveObjectsBadObjectDir(t *testing.T) {
	f := gc5NewFixture(t)

	_, err := f.svc.ProcessPreReceiveObjects(t.Context(), ProcessPreReceiveObjectsParams{
		ReadParams:            ReadParams{RepoUID: f.uid, AlternateObjectDirs: []string{filepath.Join(t.TempDir(), "x")}},
		FindLFSPointersParams: &FindLFSPointersParams{},
	})
	require.Error(t, err)
}

func TestGc5FindLFSPointers(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()
	q := gc5NewQuarantine(t, f)

	_, err := f.svc.FindLFSPointers(ctx, &FindLFSPointersParams{})
	require.Error(t, err)

	// nothing quarantined: no candidates.
	out, err := f.svc.FindLFSPointers(ctx, &FindLFSPointersParams{ReadParams: f.read()})
	require.NoError(t, err)
	require.Empty(t, out.LFSInfos)

	lfsContent, lfsOID := gc5LFSPointer()
	lfs := q.blob(ctx, t, lfsContent)
	q.blob(ctx, t, "just a text file\n")

	out, err = f.svc.FindLFSPointers(ctx, &FindLFSPointersParams{
		ReadParams: ReadParams{RepoUID: f.uid, AlternateObjectDirs: []string{q.dir}},
	})
	require.NoError(t, err)
	require.Len(t, out.LFSInfos, 1)
	require.Equal(t, lfsOID, out.LFSInfos[0].ObjID)
	require.Equal(t, lfs, out.LFSInfos[0].SHA.String())

	_, err = f.svc.FindLFSPointers(ctx, &FindLFSPointersParams{
		ReadParams: ReadParams{RepoUID: f.uid, AlternateObjectDirs: []string{filepath.Join(t.TempDir(), "x")}},
	})
	require.Error(t, err)
}

func TestGc5FindOversizeFiles(t *testing.T) {
	mk := func(c byte, size int64, typ string) parser.BatchCheckObject {
		return parser.BatchCheckObject{SHA: sha.Must(strings.Repeat(string(c), 40)), Type: typ, Size: size}
	}
	objects := []parser.BatchCheckObject{
		mk('a', 500, "blob"), mk('b', 150, "blob"), mk('c', 50, "blob"), mk('d', 5, "blob"), mk('e', 900, "commit"),
	}

	t.Run("nil params", func(t *testing.T) {
		out := findOversizeFiles(objects, nil)
		require.Empty(t, out.FileInfos)
		require.Zero(t, out.Total)
	})
	t.Run("no limits", func(t *testing.T) {
		out := findOversizeFiles(objects, &FindOversizeFilesParams{})
		require.Zero(t, out.Total)
	})
	t.Run("legacy single limit", func(t *testing.T) {
		out := findOversizeFiles(objects, &FindOversizeFilesParams{SizeLimit: 100})
		require.Equal(t, int64(2), out.Total)
		require.Len(t, out.FileInfos, 2)
	})
	t.Run("multiple limits store files at their highest limit", func(t *testing.T) {
		out := findOversizeFiles(objects, &FindOversizeFilesParams{SizeLimits: []int64{10, 100, 400}})
		require.Equal(t, int64(3), out.Total)
		require.Equal(t, int64(1), out.TotalsPerLimit[400])
		require.Equal(t, int64(1), out.TotalsPerLimit[100])
		require.Equal(t, int64(1), out.TotalsPerLimit[10])
		require.Len(t, out.FileInfos, 3)
	})
	t.Run("file infos are capped", func(t *testing.T) {
		many := make([]parser.BatchCheckObject, 0, maxOversizeFiles+5)
		for range maxOversizeFiles + 5 {
			many = append(many, mk('f', 10, "blob"))
		}
		out := findOversizeFiles(many, &FindOversizeFilesParams{SizeLimits: []int64{1}})
		require.Equal(t, int64(maxOversizeFiles+5), out.Total)
		require.Len(t, out.FileInfosPerLimit[1], maxOversizeFiles)
	})
}

func TestGc5ScanSecrets(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	_, err := f.svc.ScanSecrets(ctx, &ScanSecretsParams{})
	require.Error(t, err)

	baseRev := f.head(ctx, t).String()
	clean, err := f.svc.ScanSecrets(ctx, &ScanSecretsParams{ReadParams: f.read(), Rev: baseRev})
	require.NoError(t, err)
	require.Empty(t, clean.Findings)

	// build a github-style token at runtime so that the test source holds no scannable literal.
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var leak strings.Builder
	leak.WriteString("ghp_")
	for i := range 36 {
		leak.WriteByte(alphabet[(i*7+i/5)%len(alphabet)])
	}
	f.commit(ctx, t, CommitFileAction{
		Action: CreateAction, Path: "config.txt", Payload: []byte("github_pat = " + leak.String() + "\n"),
	})

	tip := f.head(ctx, t).String()
	out, err := f.svc.ScanSecrets(ctx, &ScanSecretsParams{ReadParams: f.read(), Rev: tip})
	require.NoError(t, err)
	require.NotEmpty(t, out.Findings)
	finding := out.Findings[0]
	require.Equal(t, "config.txt", finding.File)
	require.NotEmpty(t, finding.RuleID)
	require.NotEmpty(t, finding.Fingerprint)
	require.Equal(t, "Gc5Actor", finding.Author)

	// scanning only the clean base revision finds nothing.
	out, err = f.svc.ScanSecrets(ctx, &ScanSecretsParams{ReadParams: f.read(), Rev: baseRev})
	require.NoError(t, err)
	require.Empty(t, out.Findings)

	// base..rev range includes the leaking commit.
	out, err = f.svc.ScanSecrets(ctx, &ScanSecretsParams{ReadParams: f.read(), BaseRev: baseRev, Rev: tip})
	require.NoError(t, err)
	require.NotEmpty(t, out.Findings)

	// missing ignore file is ignored.
	out, err = f.svc.ScanSecrets(ctx, &ScanSecretsParams{
		ReadParams: f.read(), Rev: tip, GitleaksIgnorePath: DefaultGitleaksIgnorePath,
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.Findings)

	// an ignore path that points at a directory is ignored, too.
	out, err = f.svc.ScanSecrets(ctx, &ScanSecretsParams{
		ReadParams: f.read(), Rev: tip, GitleaksIgnorePath: "dir",
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.Findings)

	// a .gitleaksignore file with the fingerprint suppresses the finding.
	f.commit(ctx, t, CommitFileAction{
		Action: CreateAction, Path: DefaultGitleaksIgnorePath, Payload: []byte(finding.Fingerprint + "\n"),
	})
	tip = f.head(ctx, t).String()
	out, err = f.svc.ScanSecrets(ctx, &ScanSecretsParams{
		ReadParams: f.read(), Rev: tip, GitleaksIgnorePath: DefaultGitleaksIgnorePath,
	})
	require.NoError(t, err)
	require.Empty(t, out.Findings)
}

func TestGc5MapFindings(t *testing.T) {
	require.Empty(t, mapFindings(nil))
}

func TestGc5GeneratePipeline(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	_, err := f.svc.GeneratePipeline(ctx, &GeneratePipelineParams{})
	require.Error(t, err)

	out, err := f.svc.GeneratePipeline(ctx, &GeneratePipelineParams{ReadParams: f.read()})
	if err == nil {
		require.NotNil(t, out.PipelineYAML)
	} else {
		require.NotErrorIs(t, err, os.ErrNotExist)
	}

	_, err = f.svc.GeneratePipeline(ctx, &GeneratePipelineParams{ReadParams: ReadParams{RepoUID: "does-not-exist"}})
	require.Error(t, err)
}

func TestGc5ValidateCommitSHA(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{strings.Repeat("a", 40), true},
		{strings.Repeat("0", 64), true},
		{strings.Repeat("a", 39), false},
		{strings.Repeat("A", 40), false},
		{strings.Repeat("g", 40), false},
		{"", false},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, ValidateCommitSHA(tt.in), tt.in)
	}
}

func TestGc5StreamReader(t *testing.T) {
	data := make(chan int, 2)
	errCh := make(chan error, 1)
	data <- 1
	data <- 2
	close(data)

	r := NewStreamReader[int](data, errCh)
	v, err := r.Next()
	require.NoError(t, err)
	require.Equal(t, 1, v)
	v, err = r.Next()
	require.NoError(t, err)
	require.Equal(t, 2, v)
	_, err = r.Next()
	require.ErrorIs(t, err, io.EOF)

	// error channel delivers errors then EOF once closed.
	data2 := make(chan int)
	errCh2 := make(chan error, 1)
	errCh2 <- errors.New("boom")
	r2 := NewStreamReader[int](data2, errCh2)
	_, err = r2.Next()
	require.EqualError(t, err, "boom")
	close(errCh2)
	_, err = r2.Next()
	require.ErrorIs(t, err, io.EOF)
}

func TestGc5RefPaths(t *testing.T) {
	require.Equal(t, "refs/heads/main", GetBranchRefPath("main"))
	require.Equal(t, "refs/tags/v1", GetTagRefPath("v1"))
}
