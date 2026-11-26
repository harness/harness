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

package starlark

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func infFixtures() (*types.Repository, *types.Pipeline, *types.Execution) {
	repo := &types.Repository{
		Identifier:    "myrepo",
		Path:          "space/myrepo",
		GitURL:        "https://example.invalid/space/myrepo.git",
		DefaultBranch: "main",
	}
	pipeline := &types.Pipeline{ConfigPath: ".drone.star"}
	execution := &types.Execution{
		Event:  "push",
		Target: "feature",
		After:  "abc123",
		Debug:  true,
		Params: map[string]string{"color": "blue"},
	}
	return repo, pipeline, execution
}

func infParse(t *testing.T, script string, stepLimit, sizeLimit uint64) (string, error) {
	t.Helper()
	repo, pipeline, execution := infFixtures()
	return Parse(repo, true, pipeline, execution, &file.File{Data: []byte(script)}, stepLimit, sizeLimit)
}

func TestInfParse_MainReturningDict(t *testing.T) {
	out, err := infParse(t, `
def main(ctx):
    return {"kind": "pipeline", "branch": ctx.build.target, "slug": ctx.repo.slug}
`, 0, 0)
	require.NoError(t, err)
	require.Equal(t, `{"kind": "pipeline", "branch": "feature", "slug": "space/myrepo"}`, out)
}

func TestInfParse_MainReturningList(t *testing.T) {
	out, err := infParse(t, `
def main(ctx):
    return [{"name": "one"}, {"name": ctx.repo.name}]
`, 0, 0)
	require.NoError(t, err)
	require.Equal(t, "---\n{\"name\": \"one\"}\n---\n{\"name\": \"myrepo\"}\n", out)
}

func TestInfParse_ContextArguments(t *testing.T) {
	out, err := infParse(t, `
def main(ctx):
    return {
        "ns": ctx.repo.namespace,
        "private": ctx.repo.private,
        "event": ctx.build.event,
        "commit": ctx.build.commit,
        "debug": ctx.build.debug,
        "color": ctx.build.params["color"],
        "config": ctx.repo.config,
    }
`, 0, 0)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Equal(t, map[string]any{
		"ns":      "space",
		"private": false,
		"event":   "push",
		"commit":  "abc123",
		"debug":   true,
		"color":   "blue",
		"config":  ".drone.star",
	}, got)
}

func TestInfParse_PrivateRepoFlag(t *testing.T) {
	repo, pipeline, execution := infFixtures()
	out, err := Parse(repo, false, pipeline, execution,
		&file.File{Data: []byte("def main(ctx):\n    return {\"p\": ctx.repo.private}\n")}, 0, 0)
	require.NoError(t, err)
	require.Equal(t, `{"p": true}`, out)
}

func TestInfParse_Errors(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		stepLimit uint64
		sizeLimit uint64
		wantErr   error
		wantMsg   string
	}{
		{name: "syntax error", script: "def main(ctx:"},
		{name: "missing main", script: "x = 1", wantErr: ErrMainMissing},
		{name: "main not callable", script: "main = 1", wantErr: ErrMainInvalid},
		{
			name:    "main returns string",
			script:  "def main(ctx):\n    return 'x'\n",
			wantErr: ErrMainReturn,
		},
		{
			name:      "size limit",
			script:    "def main(ctx):\n    return {'k': 'a' * 100}\n",
			sizeLimit: 10,
			wantErr:   ErrMaximumSize,
		},
		{
			name:    "load is forbidden",
			script:  "load('other.star', 'x')\ndef main(ctx):\n    return {}\n",
			wantMsg: ErrCannotLoad.Error(),
		},
		{
			name:      "step limit",
			script:    "def main(ctx):\n    for i in range(100000):\n        pass\n    return {}\n",
			stepLimit: 100,
			wantMsg:   "too many steps",
		},
		{
			name:    "runtime error",
			script:  "def main(ctx):\n    return 1 // 0\n",
			wantMsg: "division by zero",
		},
		{
			name:    "unserialisable list item",
			script:  "def main(ctx):\n    return [len]\n",
			wantMsg: "can't be converted to JSON",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := infParse(t, tt.script, tt.stepLimit, tt.sizeLimit)
			require.Error(t, err)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantMsg != "" {
				require.Contains(t, err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestInfParse_PrintDoesNotFail(t *testing.T) {
	out, err := infParse(t, "def main(ctx):\n    print('hello')\n    return {}\n", 0, 0)
	require.NoError(t, err)
	require.Equal(t, "{}", out)
}

func TestInfWrite(t *testing.T) {
	dict := starlark.NewDict(2)
	require.NoError(t, dict.SetKey(starlark.String("a"), starlark.MakeInt(1)))
	require.NoError(t, dict.SetKey(starlark.String("b"), starlark.Bool(false)))

	tests := []struct {
		name string
		val  starlark.Value
		want string
	}{
		{"none", starlark.None, "null"},
		{"true", starlark.Bool(true), "true"},
		{"false", starlark.Bool(false), "false"},
		{"int", starlark.MakeInt(42), "42"},
		{"float", starlark.Float(1.5), "1.5"},
		{"string", starlark.String("hi"), `"hi"`},
		{"string needing json escaping", starlark.String("tab\there"), `"tab\there"`},
		{"astral string", starlark.String("\U0001F600"), "\"\U0001F600\""},
		{"list", starlark.NewList([]starlark.Value{starlark.MakeInt(1), starlark.String("x")}), `[1, "x"]`},
		{"empty list", starlark.NewList(nil), `[]`},
		{"tuple", starlark.Tuple{starlark.MakeInt(1), starlark.None}, `[1, null]`},
		{"dict", dict, `{"a": 1, "b": false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			require.NoError(t, write(buf, tt.val))
			require.Equal(t, tt.want, buf.String())
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		err := write(&bytes.Buffer{}, starlark.NewSet(0))
		require.ErrorContains(t, err, "can't be converted to JSON")
	})
	t.Run("nested unsupported", func(t *testing.T) {
		require.Error(t, write(&bytes.Buffer{}, starlark.NewList([]starlark.Value{starlark.NewSet(0)})))
		d := starlark.NewDict(1)
		require.NoError(t, d.SetKey(starlark.String("k"), starlark.NewSet(0)))
		require.Error(t, write(&bytes.Buffer{}, d))
	})
}

func TestInfIsQuoteSafe(t *testing.T) {
	require.True(t, isQuoteSafe("plain text"))
	require.False(t, isQuoteSafe("with\nnewline"))
	require.False(t, isQuoteSafe(strings.Repeat("a", 3)+"\U00010000"))
}

type infFailingWriter struct {
	failAfter int
	written   int
}

func (w *infFailingWriter) Write(p []byte) (int, error) {
	if w.written >= w.failAfter {
		return 0, bytes.ErrTooLarge
	}
	w.written += len(p)
	return len(p), nil
}

func (w *infFailingWriter) WriteByte(b byte) error {
	_, err := w.Write([]byte{b})
	return err
}

func (w *infFailingWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestInfWrite_PropagatesWriterErrors(t *testing.T) {
	d := starlark.NewDict(2)
	require.NoError(t, d.SetKey(starlark.String("a"), starlark.NewList([]starlark.Value{starlark.MakeInt(1)})))
	require.NoError(t, d.SetKey(starlark.String("b"), starlark.String("v")))

	// every prefix length of the output must surface the writer failure as an error.
	for failAfter := 0; failAfter < 14; failAfter++ {
		w := &infFailingWriter{failAfter: failAfter}
		require.Error(t, write(w, d), "failAfter=%d", failAfter)
	}
	for _, v := range []starlark.Value{
		starlark.None, starlark.MakeInt(7), starlark.String("t\tab"),
	} {
		require.Error(t, write(&infFailingWriter{}, v))
	}
}
