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

package diff

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

	"github.com/harness/gitness/git/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gtxGit(ctx context.Context, t *testing.T, dir string, args ...string) string {
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

	return stdout.String()
}

func gtxWrite(t *testing.T, dir, name, content string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

// gtxRealDiff creates two commits that exercise all kinds of file changes and returns the diff between them.
func gtxRealDiff(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	ctx := context.Background()
	dir := t.TempDir()
	gtxGit(ctx, t, dir, "init", "-q", "-b", "main")

	var long strings.Builder
	for i := 1; i <= 30; i++ {
		long.WriteString("line " + string(rune('a'+i%26)) + string(rune('a'+i/26)) + "\n")
	}

	gtxWrite(t, dir, "keep.txt", "one\ntwo\nthree\nfour\nfive\n")
	gtxWrite(t, dir, "old.txt", long.String())
	gtxWrite(t, dir, "old2.txt", long.String()+"tail\n")
	gtxWrite(t, dir, "del.txt", "to be deleted\n")
	gtxWrite(t, dir, "zz-mode.sh", "#!/bin/sh\n")
	gtxWrite(t, dir, "bin.dat", "bin\x00\x01\x02\x03")
	gtxWrite(t, dir, "space name.txt", "with space\n")
	gtxGit(ctx, t, dir, "add", "-A")
	gtxGit(ctx, t, dir, "commit", "-q", "-m", "first")

	gtxWrite(t, dir, "keep.txt", "one\nTWO\nthree\nfour\nfive\nsix\n")
	require.NoError(t, os.Rename(filepath.Join(dir, "old.txt"), filepath.Join(dir, "renamed.txt")))
	require.NoError(t, os.Remove(filepath.Join(dir, "old2.txt")))
	gtxWrite(t, dir, "renamed2.txt", long.String()+"tail changed\n")
	require.NoError(t, os.Remove(filepath.Join(dir, "del.txt")))
	gtxWrite(t, dir, "added.txt", "added 1\nadded 2\n")
	require.NoError(t, os.Chmod(filepath.Join(dir, "zz-mode.sh"), 0o700))
	gtxWrite(t, dir, "bin.dat", "bin\x00\x04\x05\x06")
	gtxWrite(t, dir, "space name.txt", "with space changed\n")
	gtxGit(ctx, t, dir, "add", "-A")
	gtxGit(ctx, t, dir, "commit", "-q", "-m", "second")

	return gtxGit(ctx, t, dir, "diff", "-M", "HEAD~1", "HEAD")
}

func gtxParse(t *testing.T, input string, includePatch bool) map[string]*File {
	t.Helper()

	p := &Parser{Reader: bufio.NewReader(strings.NewReader(input)), IncludePatch: includePatch}
	files := map[string]*File{}
	require.NoError(t, p.Parse(func(f *File) error {
		files[f.Path] = f
		return nil
	}))

	return files
}

func TestGtxParser_ParseRealDiff(t *testing.T) {
	raw := gtxRealDiff(t)

	files := gtxParse(t, raw, true)
	require.Len(t, files, 8)

	t.Run("modified file", func(t *testing.T) {
		f := files["keep.txt"]
		require.NotNil(t, f)
		assert.Equal(t, FileChange, f.Type)
		assert.Equal(t, "changed", f.Status())
		assert.Equal(t, 1, f.NumSections())
		assert.Equal(t, 2, f.NumAdditions())
		assert.Equal(t, 1, f.NumDeletions())
		assert.Equal(t, 3, f.NumChanges())
		assert.Equal(t, enum.EntryBlob, f.Mode())
		assert.Equal(t, enum.EntryBlob, f.OldMode())
		assert.NotEmpty(t, f.SHA)
		assert.NotEmpty(t, f.OldSHA)
		assert.NotEqual(t, f.SHA, f.OldSHA)
		assert.False(t, f.IsBinary)
		assert.False(t, f.IsSubmodule)
		assert.False(t, f.IsEmpty())
		assert.Contains(t, f.Patch.String(), "diff --git a/keep.txt b/keep.txt")
		assert.Contains(t, f.Patch.String(), "+TWO")

		sec := f.Sections[0]
		assert.Equal(t, sec.NumLines(), len(sec.Lines))
		assert.Equal(t, DiffLineSection, sec.Lines[0].Type)
		assert.True(t, strings.HasPrefix(sec.Lines[0].Content, "@@ -1,5 +1,6 @@"))
	})

	t.Run("added file", func(t *testing.T) {
		f := files["added.txt"]
		require.NotNil(t, f)
		assert.Equal(t, FileAdd, f.Type)
		assert.Equal(t, "added", f.Status())
		assert.Equal(t, 2, f.NumAdditions())
		assert.Zero(t, f.NumDeletions())
		assert.Equal(t, enum.EntryBlob, f.Mode())
		assert.Equal(t, enum.EntryBlob, f.OldMode())
		assert.Regexp(t, "^0+$", f.OldSHA)
	})

	t.Run("deleted file", func(t *testing.T) {
		f := files["del.txt"]
		require.NotNil(t, f)
		assert.Equal(t, FileDelete, f.Type)
		assert.Equal(t, "deleted", f.Status())
		assert.Equal(t, 1, f.NumDeletions())
		assert.Regexp(t, "^0+$", f.SHA)
	})

	t.Run("pure rename", func(t *testing.T) {
		f := files["renamed.txt"]
		require.NotNil(t, f)
		assert.Equal(t, FileRename, f.Type)
		assert.Equal(t, "renamed", f.Status())
		assert.Equal(t, "old.txt", f.OldPath)
		assert.Zero(t, f.NumSections())
	})

	t.Run("rename with changes", func(t *testing.T) {
		f := files["renamed2.txt"]
		require.NotNil(t, f)
		assert.Equal(t, FileRename, f.Type)
		assert.Equal(t, "old2.txt", f.OldPath)
		assert.Equal(t, 1, f.NumSections())
		assert.Equal(t, 1, f.NumAdditions())
		assert.Equal(t, 1, f.NumDeletions())
	})

	t.Run("mode change", func(t *testing.T) {
		f := files["zz-mode.sh"]
		require.NotNil(t, f)
		assert.Equal(t, FileChange, f.Type)
		assert.Equal(t, enum.EntryExec, f.Mode())
		assert.Equal(t, enum.EntryBlob, f.OldMode())
		assert.Zero(t, f.NumSections())
	})

	t.Run("binary file", func(t *testing.T) {
		f := files["bin.dat"]
		require.NotNil(t, f)
		assert.True(t, f.IsBinary)
		assert.Zero(t, f.NumSections())
	})

	t.Run("path with space", func(t *testing.T) {
		f := files["space name.txt"]
		require.NotNil(t, f)
		assert.Equal(t, 1, f.NumAdditions())
	})

	t.Run("patch is only collected on request", func(t *testing.T) {
		for _, f := range gtxParse(t, raw, false) {
			assert.Zero(t, f.Patch.Len(), f.Path)
		}
	})

	t.Run("nil send func", func(t *testing.T) {
		p := &Parser{Reader: bufio.NewReader(strings.NewReader(raw))}
		require.NoError(t, p.Parse(nil))
	})
}

func TestGtxParser_ParseSynthetic(t *testing.T) {
	t.Run("submodule", func(t *testing.T) {
		input := "diff --git a/sub b/sub\n" +
			"new file mode 160000\n" +
			"index 0000000..1234567\n" +
			"--- /dev/null\n" +
			"+++ b/sub\n" +
			"@@ -0,0 +1 @@\n" +
			"+Subproject commit 1234567890123456789012345678901234567890\n"

		files := gtxParse(t, input, false)
		require.Len(t, files, 1)

		f := files["sub"]
		require.NotNil(t, f)
		assert.True(t, f.IsSubmodule)
		assert.Equal(t, FileAdd, f.Type)
		assert.Equal(t, enum.EntryCommit, f.Mode())
		assert.Equal(t, "0000000", f.OldSHA)
		assert.Equal(t, "1234567", f.SHA)
	})

	t.Run("deleted submodule", func(t *testing.T) {
		input := "diff --git a/sub b/sub\n" +
			"deleted file mode 160000\n" +
			"index 1234567..0000000\n" +
			"--- a/sub\n" +
			"+++ /dev/null\n" +
			"@@ -1 +0,0 @@\n" +
			"-Subproject commit 1234567890123456789012345678901234567890\n"

		f := gtxParse(t, input, false)["sub"]
		require.NotNil(t, f)
		assert.True(t, f.IsSubmodule)
		assert.Equal(t, FileDelete, f.Type)
		assert.Equal(t, 1, f.NumDeletions())
	})

	t.Run("explicit mode lines", func(t *testing.T) {
		input := "diff --git a/x b/x\n" +
			"old mode 100644\n" +
			"new mode 100755\n" +
			"index abc..def\n"

		f := gtxParse(t, input, false)["x"]
		require.NotNil(t, f)
		assert.Equal(t, enum.EntryExec, f.Mode())
		assert.Equal(t, enum.EntryBlob, f.OldMode())
	})

	t.Run("no newline marker and empty lines inside section", func(t *testing.T) {
		input := "diff --git a/x b/x\n" +
			"index abc..def 100644\n" +
			"--- a/x\n" +
			"+++ b/x\n" +
			"@@ -1,2 +1,2 @@\n" +
			" same\n" +
			"\n" +
			"-old\n" +
			"\\ No newline at end of file\n" +
			"+new\n" +
			"\\ No newline at end of file\n"

		f := gtxParse(t, input, true)["x"]
		require.NotNil(t, f)
		require.Len(t, f.Sections, 1)

		// section line, context, delete, add
		lines := f.Sections[0].Lines
		require.Len(t, lines, 4)
		assert.Equal(t, DiffLinePlain, lines[1].Type)
		assert.Equal(t, DiffLineDelete, lines[2].Type)
		assert.Equal(t, DiffLineAdd, lines[3].Type)
		assert.Equal(t, 2, lines[2].LeftLine)
		assert.Equal(t, 2, lines[3].RightLine)
	})

	t.Run("multiple sections", func(t *testing.T) {
		input := "diff --git a/x b/x\n" +
			"index abc..def 100644\n" +
			"@@ -1 +1 @@\n" +
			"-a\n" +
			"+b\n" +
			"@@ -10 +10,2 @@\n" +
			" c\n" +
			"+d\n"

		f := gtxParse(t, input, false)["x"]
		require.NotNil(t, f)
		assert.Equal(t, 2, f.NumSections())
		assert.Equal(t, 2, f.NumAdditions())
		assert.Equal(t, 1, f.NumDeletions())
	})

	t.Run("garbage before first file is ignored", func(t *testing.T) {
		input := "garbage\n\nmore garbage\n"

		assert.Empty(t, gtxParse(t, input, false))
	})

	t.Run("quoted paths", func(t *testing.T) {
		input := "diff --git \"a/q q.txt\" \"b/q q.txt\"\n" +
			"index abc..def 100644\n"

		f := gtxParse(t, input, false)["q q.txt"]
		require.NotNil(t, f)
		assert.Equal(t, "q q.txt", f.OldPath)
	})
}

func TestGtxParser_ParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "malformed header", input: "diff --git a/x c/x\n"},
		{name: "malformed index", input: "diff --git a/x b/x\nindex abcdef 100644\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Parser{Reader: bufio.NewReader(strings.NewReader(test.input))}
			require.Error(t, p.Parse(func(*File) error { return nil }))
		})
	}

	t.Run("send error of intermediate file", func(t *testing.T) {
		input := "diff --git a/x b/x\nindex abc..def 100644\n" +
			"diff --git a/y b/y\nindex abc..def 100644\n"

		sendErr := errors.New("gtx send failure")
		p := &Parser{Reader: bufio.NewReader(strings.NewReader(input))}
		err := p.Parse(func(*File) error { return sendErr })
		require.ErrorIs(t, err, sendErr)
	})

	t.Run("send error of last file", func(t *testing.T) {
		input := "diff --git a/x b/x\nindex abc..def 100644\n"

		sendErr := errors.New("gtx send failure")
		p := &Parser{Reader: bufio.NewReader(strings.NewReader(input))}
		err := p.Parse(func(*File) error { return sendErr })
		require.ErrorIs(t, err, sendErr)
	})

	t.Run("read error", func(t *testing.T) {
		readErr := errors.New("gtx read failure")
		p := &Parser{Reader: bufio.NewReader(iotest.ErrReader(readErr))}
		require.ErrorIs(t, p.Parse(nil), readErr)
	})

	t.Run("read error within header", func(t *testing.T) {
		readErr := errors.New("gtx read failure")
		reader := &gtxFailAfterReader{data: []byte("diff --git a/x b/x\n"), err: readErr}
		p := &Parser{Reader: bufio.NewReader(reader)}
		require.ErrorIs(t, p.Parse(func(*File) error { return nil }), readErr)
	})

	t.Run("read error within section", func(t *testing.T) {
		readErr := errors.New("gtx read failure")
		reader := &gtxFailAfterReader{
			data: []byte("diff --git a/x b/x\nindex abc..def 100644\n@@ -1 +1 @@\n a\n"),
			err:  readErr,
		}
		p := &Parser{Reader: bufio.NewReader(reader)}
		require.ErrorIs(t, p.Parse(func(*File) error { return nil }), readErr)
	})
}

// gtxFailAfterReader returns the data and then the error.
type gtxFailAfterReader struct {
	data []byte
	err  error
}

func (r *gtxFailAfterReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}

	n := copy(p, r.data)
	r.data = r.data[n:]

	return n, nil
}

func TestGtxFile_StatusAndCounters(t *testing.T) {
	tests := []struct {
		fileType FileType
		want     string
	}{
		{FileAdd, "added"},
		{FileChange, "changed"},
		{FileDelete, "deleted"},
		{FileRename, "renamed"},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, (&File{Type: test.fileType}).Status())
	}

	f := &File{numAdditions: 3, numDeletions: 2, Sections: []*Section{{}, {}}}
	assert.Equal(t, 5, f.NumChanges())
	assert.Equal(t, 2, f.NumSections())
}

func TestGtxFile_StatusUnchanged(t *testing.T) {
	f := &File{Type: FileType(200)}
	assert.Equal(t, "unchanged", f.Status())
	assert.True(t, (&File{}).IsEmpty())
	assert.False(t, (&File{OldPath: "x"}).IsEmpty())
}

func TestGtxSection_Line(t *testing.T) {
	// a replaced line in the middle of a file: line 2 deleted, line 2 added, context around.
	replaced := &Section{Lines: []*Line{
		{Type: DiffLineSection, Content: "@@ -1,3 +1,3 @@"},
		{Type: DiffLinePlain, Content: " a", LeftLine: 1, RightLine: 1},
		{Type: DiffLineDelete, Content: "-b", LeftLine: 2},
		{Type: DiffLineAdd, Content: "+B", RightLine: 2},
		{Type: DiffLinePlain, Content: " c", LeftLine: 3, RightLine: 3},
	}}

	// only an addition: the numbers of added and deleted lines differ.
	added := &Section{Lines: []*Line{
		{Type: DiffLineSection, Content: "@@ -1 +1,2 @@"},
		{Type: DiffLinePlain, Content: " a", LeftLine: 1, RightLine: 1},
		{Type: DiffLineAdd, Content: "+b", RightLine: 2},
	}}

	tests := []struct {
		name     string
		section  *Section
		lineType LineType
		line     int
		want     string
	}{
		{name: "deleted line found", section: replaced, lineType: DiffLineDelete, line: 2, want: "-b"},
		{name: "added line found", section: replaced, lineType: DiffLineAdd, line: 2, want: "+B"},
		{name: "deleted line not found", section: replaced, lineType: DiffLineDelete, line: 9},
		{name: "added line not found", section: replaced, lineType: DiffLineAdd, line: 9},
		{name: "plain lines are not returned", section: replaced, lineType: DiffLinePlain, line: 1},
		{name: "unbalanced section", section: added, lineType: DiffLineAdd, line: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.section.Line(test.lineType, test.line)
			if test.want == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, test.want, got.Content)
		})
	}
}
