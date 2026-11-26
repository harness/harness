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

package foreachref_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/harness/gitness/git/api/foreachref"

	"github.com/stretchr/testify/require"
)

func TestGc5FormatFlag(t *testing.T) {
	format := foreachref.NewFormat("refname", "objectname")

	require.Equal(t, "refname %(refname)%00objectname %(objectname)%00%00", format.Flag())
	require.Equal(t, "refname %(refname)%00%00", foreachref.NewFormat("refname").Flag())
}

func TestGc5ParserParsesRefs(t *testing.T) {
	format := foreachref.NewFormat("refname", "objectname", "subject")
	input := "refname refs/heads/main\x00objectname abc123\x00subject first commit\x00\x00\n" +
		"refname refs/heads/dev\x00objectname def456\x00subject \x00\x00\n"

	p := format.Parser(strings.NewReader(input))

	first := p.Next()
	require.Equal(t, map[string]string{
		"refname": "refs/heads/main", "objectname": "abc123", "subject": "first commit",
	}, first)

	second := p.Next()
	require.Equal(t, "refs/heads/dev", second["refname"])
	require.Equal(t, "def456", second["objectname"])
	require.Empty(t, second["subject"])

	require.Nil(t, p.Next())
	require.NoError(t, p.Err())
}

func TestGc5ParserEmptyInput(t *testing.T) {
	p := foreachref.NewFormat("refname").Parser(strings.NewReader(""))

	require.Nil(t, p.Next())
	require.NoError(t, p.Err())
}

func TestGc5ParserLastBlockWithoutDelimiter(t *testing.T) {
	p := foreachref.NewFormat("refname").Parser(strings.NewReader("refname refs/tags/v1"))

	require.Equal(t, map[string]string{"refname": "refs/tags/v1"}, p.Next())
	require.Nil(t, p.Next())
	require.NoError(t, p.Err())
}

func TestGc5ParserErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "wrong number of fields",
			input:   "refname refs/heads/main\x00\x00\n",
			wantErr: "unexpected number of reference fields",
		},
		{
			name:    "wrong field name",
			input:   "refname refs/heads/main\x00other abc\x00\x00\n",
			wantErr: "unexpected field name at position 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := foreachref.NewFormat("refname", "objectname").Parser(strings.NewReader(tt.input))

			require.Nil(t, p.Next())
			require.ErrorContains(t, p.Err(), tt.wantErr)
		})
	}
}

func TestGc5ParserWithGitForEachRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)

		return string(out)
	}

	run("init", "-q", "--initial-branch", "main")
	run("-c", "user.name=Gc5", "-c", "user.email=gc5@example.com", "commit", "-q", "--allow-empty", "-m", "subject line")
	run("branch", "other")

	format := foreachref.NewFormat("refname", "objectname", "subject")
	out := run("for-each-ref", "--format", format.Flag(), "refs/heads")

	p := format.Parser(strings.NewReader(out))
	var names []string
	for ref := p.Next(); ref != nil; ref = p.Next() {
		names = append(names, ref["refname"])
		require.Len(t, ref["objectname"], 40)
		require.Equal(t, "subject line", ref["subject"])
	}
	require.NoError(t, p.Err())
	require.ElementsMatch(t, []string{"refs/heads/main", "refs/heads/other"}, names)
}
