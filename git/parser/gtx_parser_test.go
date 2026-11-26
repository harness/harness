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

package parser

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gtxZeroSHA = "0000000000000000000000000000000000000000"
	gtxShaA    = "1111111111111111111111111111111111111111"
	gtxShaB    = "2222222222222222222222222222222222222222"
)

func TestGtxDiffStatus_String(t *testing.T) {
	tests := map[DiffStatus]string{
		DiffStatusModified: "M",
		DiffStatusAdded:    "A",
		DiffStatusDeleted:  "D",
		DiffStatusRenamed:  "R",
		DiffStatusCopied:   "C",
		DiffStatusType:     "T",
	}

	for status, want := range tests {
		assert.Equal(t, want, status.String())
	}
}

func TestGtxDiffRaw(t *testing.T) {
	entry := func(oldMode, newMode, oldSHA, newSHA, status string) string {
		return ":" + oldMode + " " + newMode + " " + oldSHA + " " + newSHA + " " + status
	}

	tests := []struct {
		name    string
		input   string
		want    []DiffRawFile
		wantErr string
	}{
		{name: "empty", input: "", want: nil},
		{name: "lines that don't match are skipped", input: "garbage\x00more\x00", want: nil},
		{
			name:  "modified",
			input: entry("100644", "100644", gtxShaA, gtxShaB, "M") + "\x00a.txt\x00",
			want: []DiffRawFile{{
				OldFileMode: "100644", NewFileMode: "100644", OldBlobSHA: gtxShaA, NewBlobSHA: gtxShaB,
				Status: DiffStatusModified, Path: "a.txt",
			}},
		},
		{
			name:  "added",
			input: entry("000000", "100644", gtxZeroSHA, gtxShaB, "A") + "\x00dir/new.txt\x00",
			want: []DiffRawFile{{
				OldFileMode: "000000", NewFileMode: "100644", OldBlobSHA: gtxZeroSHA, NewBlobSHA: gtxShaB,
				Status: DiffStatusAdded, Path: "dir/new.txt",
			}},
		},
		{
			name:  "deleted",
			input: entry("100644", "000000", gtxShaA, gtxZeroSHA, "D") + "\x00gone.txt\x00",
			want: []DiffRawFile{{
				OldFileMode: "100644", NewFileMode: "000000", OldBlobSHA: gtxShaA, NewBlobSHA: gtxZeroSHA,
				Status: DiffStatusDeleted, Path: "gone.txt",
			}},
		},
		{
			name:  "type change",
			input: entry("100644", "120000", gtxShaA, gtxShaB, "T") + "\x00link\x00",
			want: []DiffRawFile{{
				OldFileMode: "100644", NewFileMode: "120000", OldBlobSHA: gtxShaA, NewBlobSHA: gtxShaB,
				Status: DiffStatusType, Path: "link",
			}},
		},
		{
			name:  "renamed with similarity",
			input: entry("100644", "100644", gtxShaA, gtxShaB, "R095") + "\x00old.txt\x00new.txt\x00",
			want: []DiffRawFile{{
				OldFileMode: "100644", NewFileMode: "100644", OldBlobSHA: gtxShaA, NewBlobSHA: gtxShaB,
				Status: DiffStatusRenamed, OldPath: "old.txt", Path: "new.txt",
			}},
		},
		{
			name:  "copied",
			input: entry("100644", "100755", gtxShaA, gtxShaB, "C100") + "\x00src.sh\x00copy.sh\x00",
			want: []DiffRawFile{{
				OldFileMode: "100644", NewFileMode: "100755", OldBlobSHA: gtxShaA, NewBlobSHA: gtxShaB,
				Status: DiffStatusCopied, OldPath: "src.sh", Path: "copy.sh",
			}},
		},
		{
			name: "multiple entries",
			input: entry("100644", "100644", gtxShaA, gtxShaB, "M") + "\x00a.txt\x00" +
				entry("100644", "100644", gtxShaA, gtxShaB, "R100") + "\x00b.txt\x00c.txt\x00" +
				entry("000000", "100644", gtxZeroSHA, gtxShaB, "A") + "\x00d.txt",
			want: []DiffRawFile{
				{
					OldFileMode: "100644", NewFileMode: "100644", OldBlobSHA: gtxShaA, NewBlobSHA: gtxShaB,
					Status: DiffStatusModified, Path: "a.txt",
				},
				{
					OldFileMode: "100644", NewFileMode: "100644", OldBlobSHA: gtxShaA, NewBlobSHA: gtxShaB,
					Status: DiffStatusRenamed, OldPath: "b.txt", Path: "c.txt",
				},
				{
					OldFileMode: "000000", NewFileMode: "100644", OldBlobSHA: gtxZeroSHA, NewBlobSHA: gtxShaB,
					Status: DiffStatusAdded, Path: "d.txt",
				},
			},
		},
		{
			name:    "missing path",
			input:   entry("100644", "100644", gtxShaA, gtxShaB, "M") + "\x00",
			wantErr: "failed to get path",
		},
		{
			name:    "missing new path for rename",
			input:   entry("100644", "100644", gtxShaA, gtxShaB, "R100") + "\x00old.txt\x00",
			wantErr: "failed to get new path",
		},
		{
			name:    "invalid status",
			input:   entry("100644", "100644", gtxShaA, gtxShaB, "X") + "\x00a.txt\x00",
			wantErr: "invalid raw diff status",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DiffRaw(strings.NewReader(test.input))
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}

	t.Run("read error", func(t *testing.T) {
		_, err := DiffRaw(iotest.ErrReader(errors.New("gtx read failure")))
		require.ErrorContains(t, err, "gtx read failure")
	})
}

func gtxGit(ctx context.Context, t *testing.T, dir string, args ...string) []byte {
	t.Helper()

	full := append([]string{
		"-c", "user.name=Gtx Tester",
		"-c", "user.email=gtx@example.com",
		"-c", "commit.gpgsign=false",
	}, args...)

	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "git %v failed: %s", args, stderr.String())

	return stdout.Bytes()
}

func TestGtxDiffRaw_RealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	ctx := context.Background()
	dir := t.TempDir()
	gtxGit(ctx, t, dir, "init", "-q", "-b", "main")

	var body strings.Builder
	for i := range 20 {
		body.WriteString("some line number " + strings.Repeat("x", i) + "\n")
	}

	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

	write("mod.txt", "one\n")
	write("del.txt", "delete me\n")
	write("old.txt", body.String())
	gtxGit(ctx, t, dir, "add", "-A")
	gtxGit(ctx, t, dir, "commit", "-q", "-m", "first")

	write("mod.txt", "two\n")
	write("add.txt", "added\n")
	require.NoError(t, os.Remove(filepath.Join(dir, "del.txt")))
	require.NoError(t, os.Rename(filepath.Join(dir, "old.txt"), filepath.Join(dir, "new name.txt")))
	gtxGit(ctx, t, dir, "add", "-A")
	gtxGit(ctx, t, dir, "commit", "-q", "-m", "second")

	t.Run("raw diff", func(t *testing.T) {
		out := gtxGit(ctx, t, dir, "diff", "--raw", "-z", "-M", "HEAD~1", "HEAD")

		files, err := DiffRaw(bytes.NewReader(out))
		require.NoError(t, err)

		byPath := map[string]DiffRawFile{}
		for _, f := range files {
			byPath[f.Path] = f
		}
		require.Len(t, byPath, 4)

		assert.Equal(t, DiffStatusModified, byPath["mod.txt"].Status)
		assert.Equal(t, DiffStatusAdded, byPath["add.txt"].Status)
		assert.Equal(t, DiffStatusDeleted, byPath["del.txt"].Status)
		assert.Equal(t, DiffStatusRenamed, byPath["new name.txt"].Status)
		assert.Equal(t, "old.txt", byPath["new name.txt"].OldPath)
	})

	t.Run("cat-file batch check", func(t *testing.T) {
		out := gtxGit(ctx, t, dir, "cat-file", "--batch-all-objects", "--batch-check", "-Z")

		objects, err := CatFileBatchCheckAllObjects(bytes.NewReader(out))
		require.NoError(t, err)

		types := map[string]int{}
		for _, o := range objects {
			types[o.Type]++
			assert.False(t, o.SHA.IsEmpty())
		}
		assert.Equal(t, 2, types["commit"])
		assert.Positive(t, types["tree"])
		assert.Positive(t, types["blob"])
	})
}

func TestGtxCatFileBatchCheckAllObjects(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []BatchCheckObject
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "objects",
			input: gtxShaA + " blob 12\x00" + gtxShaB + " commit 234\x00",
			want: []BatchCheckObject{
				{SHA: sha.Must(gtxShaA), Type: "blob", Size: 12},
				{SHA: sha.Must(gtxShaB), Type: "commit", Size: 234},
			},
		},
		{
			name:  "sha-256 object",
			input: strings.Repeat("a", 64) + " tree 0\x00",
			want:  []BatchCheckObject{{SHA: sha.Must(strings.Repeat("a", 64)), Type: "tree", Size: 0}},
		},
		{name: "missing object", input: gtxShaA + " missing\x00", wantErr: true},
		{name: "garbage", input: "not an object line\x00", wantErr: true},
		{name: "size overflow", input: gtxShaA + " blob 99999999999999999999\x00", wantErr: true},
		{name: "uppercase sha", input: strings.Repeat("A", 40) + " blob 1\x00", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CatFileBatchCheckAllObjects(strings.NewReader(test.input))
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}

	t.Run("read error", func(t *testing.T) {
		_, err := CatFileBatchCheckAllObjects(iotest.ErrReader(errors.New("gtx read failure")))
		require.ErrorContains(t, err, "gtx read failure")
	})
}

func TestGtxLFSPointer(t *testing.T) {
	oid := strings.Repeat("ab12", 16)
	valid := "version https://git-lfs.github.com/spec/v1\noid sha256:" + oid + "\nsize 12345\n"

	tests := []struct {
		name     string
		content  string
		size     int64
		wantOID  string
		wantSize int64
		wantOK   bool
	}{
		{name: "valid", content: valid, size: int64(len(valid)), wantOID: oid, wantSize: 12345, wantOK: true},
		{
			name: "valid without trailing newline", content: strings.TrimSuffix(valid, "\n"),
			size: 100, wantOID: oid, wantSize: 12345, wantOK: true,
		},
		{name: "too large", content: valid, size: LfsPointerMaxSize + 1},
		{name: "at the size limit", content: valid, size: LfsPointerMaxSize, wantOID: oid, wantSize: 12345, wantOK: true},
		{name: "wrong prefix", content: "something else\noid sha256:" + oid + "\nsize 1\n", size: 50},
		{name: "empty", content: "", size: 0},
		{
			name:    "missing oid",
			content: "version https://git-lfs.github.com/spec/v1\nsize 1\n", size: 50,
		},
		{
			name:    "short oid",
			content: "version https://git-lfs.github.com/spec/v1\noid sha256:abcd\nsize 1\n", size: 50,
		},
		{
			name:    "uppercase oid",
			content: "version https://git-lfs.github.com/spec/v1\noid sha256:" + strings.ToUpper(oid) + "\nsize 1\n", size: 50,
		},
		{
			name:    "missing size",
			content: "version https://git-lfs.github.com/spec/v1\noid sha256:" + oid + "\n", size: 50,
		},
		{
			name:    "size overflow",
			content: "version https://git-lfs.github.com/spec/v1\noid sha256:" + oid + "\nsize 99999999999999999999\n",
			size:    100,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ptr, ok := IsLFSPointer(context.Background(), []byte(test.content), test.size)
			assert.Equal(t, test.wantOK, ok)
			if !test.wantOK {
				assert.Nil(t, ptr)
				return
			}

			require.NotNil(t, ptr)
			assert.Equal(t, test.wantOID, ptr.OID)
			assert.Equal(t, test.wantSize, ptr.Size)

			gotOID, err := GetLFSObjectID([]byte(test.content))
			require.NoError(t, err)
			assert.Equal(t, test.wantOID, gotOID)
		})
	}

	t.Run("get object id errors", func(t *testing.T) {
		for _, content := range []string{
			"",
			"nope",
			"version https://git-lfs.github.com/spec/v1\nsize 1\n",
			"version https://git-lfs.github.com/spec/v1\noid sha256:" + oid + "\n",
		} {
			_, err := GetLFSObjectID([]byte(content))
			require.ErrorIs(t, err, ErrInvalidLFSPointer, content)
		}
	})
}

func TestGtxScanZeroSeparated(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty", input: "", want: nil},
		{name: "single without separator", input: "abc", want: []string{"abc"}},
		{name: "single with separator", input: "abc\x00", want: []string{"abc"}},
		{name: "multiple", input: "a\x00bb\x00ccc", want: []string{"a", "bb", "ccc"}},
		{name: "empty tokens", input: "\x00\x00a\x00", want: []string{"", "", "a"}},
		{name: "newlines are not separators", input: "a\nb\x00c", want: []string{"a\nb", "c"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scan := bufio.NewScanner(strings.NewReader(test.input))
			scan.Split(ScanZeroSeparated)

			var got []string
			for scan.Scan() {
				got = append(got, scan.Text())
			}
			require.NoError(t, scan.Err())
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGtxHunkHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    HunkHeader
		wantZero  bool
		wantValid bool
		wantStr   string
	}{
		{name: "zero", header: HunkHeader{}, wantZero: true, wantStr: "@@ -0,0 +0,0 @@"},
		{
			name: "spans of one are omitted", header: HunkHeader{OldLine: 3, OldSpan: 1, NewLine: 3, NewSpan: 1},
			wantValid: true, wantStr: "@@ -3 +3 @@",
		},
		{
			name: "with spans and text", header: HunkHeader{OldLine: 1, OldSpan: 5, NewLine: 2, NewSpan: 7, Text: "func main()"},
			wantValid: true, wantStr: "@@ -1,5 +2,7 @@ func main()",
		},
		{
			name: "new file", header: HunkHeader{OldLine: 0, OldSpan: 0, NewLine: 1, NewSpan: 3},
			wantValid: true, wantStr: "@@ -0,0 +1,3 @@",
		},
		{
			name: "deleted file", header: HunkHeader{OldLine: 1, OldSpan: 3, NewLine: 0, NewSpan: 0},
			wantValid: true, wantStr: "@@ -1,3 +0,0 @@",
		},
		{
			name: "old span without line", header: HunkHeader{OldLine: 0, OldSpan: 2, NewLine: 1, NewSpan: 1},
			wantStr: "@@ -0,2 +1 @@",
		},
		{
			name: "new line without span", header: HunkHeader{OldLine: 1, OldSpan: 1, NewLine: 2, NewSpan: 0},
			wantStr: "@@ -1 +2,0 @@",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantZero, test.header.IsZero())
			assert.Equal(t, test.wantValid, test.header.IsValid())
			assert.Equal(t, test.wantStr, test.header.String())
		})
	}
}

func TestGtxDiffExtHeaderParseIndex(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNew string
		wantOld string
		wantOK  bool
	}{
		{
			name:    "with mode",
			line:    "index 68233d6cd204b0df84e91a1ce8c8b75e13529973..e69de29bb2d1d6434b8b29ae775ad8c2e48c5391 100644",
			wantNew: "68233d6cd204b0df84e91a1ce8c8b75e13529973",
			wantOld: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
			wantOK:  true,
		},
		{name: "without mode", line: "index abcd..1234", wantNew: "abcd", wantOld: "1234", wantOK: true},
		{name: "too short", line: "index abc..123"},
		{name: "uppercase", line: "index ABCD..1234"},
		{name: "not an index header", line: "similarity index 100%"},
		{name: "empty", line: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newSHA, oldSHA, ok := DiffExtHeaderParseIndex(test.line)
			assert.Equal(t, test.wantOK, ok)
			assert.Equal(t, test.wantNew, newSHA)
			assert.Equal(t, test.wantOld, oldSHA)
		})
	}
}

func TestGtxHasLineEnding(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantAny  bool
		wantLF   bool
		wantCRLF bool
	}{
		{name: "empty", line: ""},
		{name: "no ending", line: "abc"},
		{name: "lf", line: "abc\n", wantAny: true, wantLF: true},
		{name: "crlf", line: "abc\r\n", wantAny: true, wantLF: true, wantCRLF: true},
		{name: "only lf", line: "\n", wantAny: true, wantLF: true},
		{name: "cr alone", line: "abc\r"},
		{name: "lf in the middle", line: "a\nb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantAny, HasLineEnding([]byte(test.line)))
			assert.Equal(t, test.wantLF, HasLineEndingLF([]byte(test.line)))
			assert.Equal(t, test.wantCRLF, HasLineEndingCRLF([]byte(test.line)))
		})
	}
}

func TestGtxLimitLineLen(t *testing.T) {
	tests := []struct {
		name   string
		in     []string
		maxLen int
		want   []string
	}{
		{name: "no lines", in: nil, maxLen: 3, want: nil},
		{name: "short lines are kept", in: []string{"", "ab", "abc"}, maxLen: 3, want: []string{"", "ab", "abc"}},
		{name: "long line is cut", in: []string{"abcd", "xy"}, maxLen: 3, want: []string{"abc…", "xy"}},
		{name: "multi byte runes are counted once", in: []string{"ééééé"}, maxLen: 2, want: []string{"éé…"}},
		{name: "multi byte at the limit", in: []string{"日本語"}, maxLen: 3, want: []string{"日本語"}},
		{name: "zero limit", in: []string{"a"}, maxLen: 0, want: []string{"…"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lines := test.in
			LimitLineLen(&lines, test.maxLen)
			assert.Equal(t, test.want, lines)
		})
	}
}
