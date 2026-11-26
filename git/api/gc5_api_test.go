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
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func gc5Git(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Gc5One", "GIT_AUTHOR_EMAIL=gc5-one@example.com",
		"GIT_COMMITTER_NAME=Gc5One", "GIT_COMMITTER_EMAIL=gc5-one@example.com",
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=maintenance.auto", "GIT_CONFIG_VALUE_0=false",
		"GIT_CONFIG_KEY_1=gc.auto", "GIT_CONFIG_VALUE_1=0",
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)

	return strings.TrimSpace(string(out))
}

func gc5Write(t *testing.T, repo, rel, content string, mode os.FileMode) {
	t.Helper()

	full := filepath.Join(repo, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), mode))
}

// gc5Repo creates a repository with three commits: the second one changes a line of file.txt (committed by a second
// author), the third adds an executable script, a symlink and a submodule entry.
func gc5Repo(t *testing.T) (repo, first, second string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	repo = t.TempDir()
	gc5Git(t, repo, "init", "-q", "--initial-branch", "main")

	gc5Write(t, repo, "file.txt", "line one\nline two\nline three\n", 0o644)
	gc5Git(t, repo, "add", ".")
	gc5Git(t, repo, "commit", "-q", "-m", "first")
	first = gc5Git(t, repo, "rev-parse", "HEAD")

	gc5Write(t, repo, "file.txt", "line one\nline 2\nline three\n", 0o644)
	gc5Git(t, repo, "add", ".")
	gc5Git(t, repo, "-c", "user.name=Gc5Two", "-c", "user.email=gc5-two@example.com",
		"commit", "-q", "--author=Gc5Two <gc5-two@example.com>", "-m", "second")
	second = gc5Git(t, repo, "rev-parse", "HEAD")

	gc5Write(t, repo, "bin/run.sh", "#!/bin/sh\n", 0o755)
	require.NoError(t, os.Symlink("file.txt", filepath.Join(repo, "link")))
	gc5Git(t, repo, "add", ".")
	gc5Git(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+first+",sub")
	gc5Git(t, repo, "commit", "-q", "-m", "third")

	return repo, first, second
}

func TestGc5PacketWrite(t *testing.T) {
	require.Equal(t, "0006a\n", string(packetWrite("a\n")))
	require.Equal(t, "001e# service=git-upload-pack\n", string(packetWrite("# service=git-upload-pack\n")))
	require.Equal(t, "0004", string(packetWrite("")))
}

func TestGc5PktError(t *testing.T) {
	var buf bytes.Buffer

	PktError(&buf, errors.New("boom"))

	require.Equal(t, "000dERR boom\n", buf.String())
}

func TestGc5InfoRefs(t *testing.T) {
	repo, _, _ := gc5Repo(t)
	g := &Git{}

	var buf bytes.Buffer
	require.NoError(t, g.InfoRefs(t.Context(), repo, "upload-pack", &buf))
	out := buf.String()
	require.True(t, strings.HasPrefix(out, "001e# service=git-upload-pack\n0000"), out)
	require.Contains(t, out, "refs/heads/main")

	buf.Reset()
	require.NoError(t, g.InfoRefs(t.Context(), repo, "upload-pack", &buf, "GIT_PROTOCOL=version=2"))
	require.Contains(t, buf.String(), "version 2")

	require.Error(t, g.InfoRefs(t.Context(), repo, "no-such-service", io.Discard))
	require.Error(t, g.InfoRefs(t.Context(), filepath.Join(repo, "missing"), "upload-pack", io.Discard))
}

type gc5FailWriter struct{ failAfter, writes int }

func (w *gc5FailWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > w.failAfter {
		return 0, errors.New("write failed")
	}

	return len(p), nil
}

func TestGc5InfoRefsWriteErrors(t *testing.T) {
	repo, _, _ := gc5Repo(t)
	g := &Git{}

	for failAfter := range 3 {
		err := g.InfoRefs(t.Context(), repo, "upload-pack", &gc5FailWriter{failAfter: failAfter})
		require.Error(t, err, "failAfter=%d", failAfter)
	}
}

func TestGc5ServicePack(t *testing.T) {
	repo, _, _ := gc5Repo(t)
	g := &Git{}

	t.Run("stateless advertisement", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := g.ServicePack(t.Context(), repo, ServicePackOptions{
			Service:      enum.GitServiceTypeUploadPack,
			StatelessRPC: true,
			Stdin:        strings.NewReader("0000"),
			Stdout:       &stdout,
			Stderr:       &stderr,
			Protocol:     "version=2",
		})
		require.NoError(t, err)
		_ = stdout
	})

	t.Run("unsafe protocol header and hook are tolerated", func(t *testing.T) {
		err := g.ServicePack(t.Context(), repo, ServicePackOptions{
			Service:      enum.GitServiceTypeUploadPack,
			StatelessRPC: true,
			Stdin:        strings.NewReader("0000"),
			Stdout:       io.Discard,
			Stderr:       io.Discard,
			Protocol:     "bad header;rm -rf",
			Config:       ServicePackConfig{UploadPackHook: "true"},
		})
		require.NoError(t, err)
	})

	t.Run("unknown object fails", func(t *testing.T) {
		err := g.ServicePack(t.Context(), repo, ServicePackOptions{
			Service:      enum.GitServiceTypeUploadPack,
			StatelessRPC: true,
			Stdin:        strings.NewReader("0032want " + strings.Repeat("1", 40) + "\n00000009done\n"),
			Stdout:       io.Discard,
			Stderr:       io.Discard,
		})
		require.Error(t, err)
	})

	t.Run("cancelled run is a warning", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := g.ServicePack(ctx, repo, ServicePackOptions{
			Service:      enum.GitServiceTypeUploadPack,
			StatelessRPC: true,
			Stdin:        strings.NewReader("0000"),
			Stdout:       io.Discard,
			Stderr:       io.Discard,
		})
		require.Error(t, err)
	})
}

func TestGc5IsCallerCancel(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	require.True(t, isCallerCancel(cancelled, context.Canceled))
	require.True(t, isCallerCancel(cancelled, errors.New("other")))
	require.False(t, isCallerCancel(t.Context(), errors.New("other")))

	// cancellation right at the deadline counts as the deadline firing.
	nearDeadline, cancelNear := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancelNear()
	<-nearDeadline.Done()
	require.False(t, isCallerCancel(nearDeadline, context.Canceled))

	farDeadline, cancelFar := context.WithTimeout(t.Context(), time.Hour)
	cancelFar()
	require.True(t, isCallerCancel(farDeadline, context.Canceled))
}

func TestGc5Blame(t *testing.T) {
	repo, first, second := gc5Repo(t)
	g := &Git{}

	reader := g.Blame(t.Context(), repo, "main", "file.txt", 0, 0)

	var parts []*BlamePart
	for {
		part, err := reader.NextPart()
		// the last part is returned together with io.EOF.
		if part != nil {
			parts = append(parts, part)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.Len(t, parts, 3)
	require.Equal(t, first, parts[0].Commit.SHA.String())
	require.Equal(t, second, parts[1].Commit.SHA.String())
	require.Equal(t, []string{"line 2"}, parts[1].Lines)
	require.Equal(t, "Gc5Two", parts[1].Commit.Author.Identity.Name)
	require.NotNil(t, parts[1].Previous)
	require.Equal(t, first, parts[1].Previous.CommitSHA.String())

	// restrict to a line range.
	ranged := g.Blame(t.Context(), repo, "main", "file.txt", 2, 2)
	part, err := ranged.NextPart()
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, []string{"line 2"}, part.Lines)
	part, err = ranged.NextPart()
	require.ErrorIs(t, err, io.EOF)
	require.Nil(t, part)

	// open ended range.
	open := g.Blame(t.Context(), repo, "main", "file.txt", 3, 0)
	part, err = open.NextPart()
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, []string{"line three"}, part.Lines)

	// unknown file.
	missing := g.Blame(t.Context(), repo, "main", "missing.txt", 0, 0)
	_, err = missing.NextPart()
	require.Error(t, err)
	require.NotErrorIs(t, err, io.EOF)

	// range beyond the end of the file.
	beyond := g.Blame(t.Context(), repo, "main", "file.txt", 50, 60)
	_, err = beyond.NextPart()
	require.Error(t, err)
}

func TestGc5FSNodes(t *testing.T) {
	repo, first, _ := gc5Repo(t)
	gfs := NewFS(t.Context(), "main", repo)

	read := func(name string) string {
		t.Helper()
		data, err := fs.ReadFile(gfs, name)
		require.NoError(t, err)

		return string(data)
	}
	require.Equal(t, "line one\nline 2\nline three\n", read("file.txt"))

	// regular file info.
	f, err := gfs.Open("file.txt")
	require.NoError(t, err)
	info, err := f.Stat()
	require.NoError(t, err)
	require.Equal(t, "file.txt", info.Name())
	require.Equal(t, int64(len("line one\nline 2\nline three\n")), info.Size())
	require.Equal(t, fs.FileMode(0o444), info.Mode())
	require.NoError(t, f.Close())

	// executable file.
	f, err = gfs.Open("bin/run.sh")
	require.NoError(t, err)
	info, err = f.Stat()
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0o555), info.Mode())
	require.NoError(t, f.Close())

	// symlink.
	f, err = gfs.Open("link")
	require.NoError(t, err)
	info, err = f.Stat()
	require.NoError(t, err)
	require.Equal(t, fs.ModeSymlink, info.Mode())
	require.NoError(t, f.Close())

	// submodule: content is the commit sha plus newline.
	f, err = gfs.Open("sub")
	require.NoError(t, err)
	info, err = f.Stat()
	require.NoError(t, err)
	require.Equal(t, fs.ModeIrregular, info.Mode())
	require.Equal(t, int64(len(first)+1), info.Size())
	require.Equal(t, "sub", info.Name())
	content, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, first+"\n", string(content))
	require.NoError(t, f.Close())

	// directory info.
	d, err := gfs.Open("bin")
	require.NoError(t, err)
	dirInfo, err := d.Stat()
	require.NoError(t, err)
	require.True(t, dirInfo.IsDir())
	require.Equal(t, fs.ModeDir, dirInfo.Mode())
	require.Zero(t, dirInfo.Size())
	require.Equal(t, "bin", dirInfo.Name())
	require.NoError(t, d.Close())

	// errors.
	_, err = gfs.Open("../escape")
	require.ErrorIs(t, err, fs.ErrInvalid)
	_, err = gfs.Open("does/not/exist")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestGc5GetCommit(t *testing.T) {
	repo, _, second := gc5Repo(t)

	commit, err := GetCommit(t.Context(), repo, sha.Must(second))
	require.NoError(t, err)
	require.Equal(t, "second", commit.Title)
	require.Equal(t, "Gc5Two", commit.Author.Identity.Name)

	_, err = GetCommit(t.Context(), repo, sha.Must(strings.Repeat("1", 40)))
	require.Error(t, err)
}
