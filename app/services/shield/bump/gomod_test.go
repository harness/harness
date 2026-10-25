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

package bump

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"

	"github.com/stretchr/testify/require"
)

const testMod = "example.com/vuln"

// fakeProxy serves the given module versions: version -> go.mod content.
func fakeProxy(t *testing.T, mods map[string]string) *GoProxy {
	t.Helper()
	zips := map[string][]byte{}
	for v, gomod := range mods {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package vuln\n"), 0o600))
		var buf bytes.Buffer
		require.NoError(t, modzip.CreateFromDir(&buf, module.Version{Path: testMod, Version: v}, dir))
		zips[v] = buf.Bytes()
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest, ok := strings.CutPrefix(r.URL.Path, "/"+testMod+"/@v/")
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch {
		case strings.HasSuffix(rest, ".mod"):
			if m, ok := mods[strings.TrimSuffix(rest, ".mod")]; ok {
				_, _ = w.Write([]byte(m))
				return
			}
		case strings.HasSuffix(rest, ".zip"):
			if z, ok := zips[strings.TrimSuffix(rest, ".zip")]; ok {
				_, _ = w.Write(z)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewGoProxy(srv.URL)
}

const projectMod = `module example.com/app

go 1.21

require (
	example.com/vuln v1.2.0
	example.com/other v1.0.0 // indirect
)
`

func TestBumpGoMod(t *testing.T) {
	proxy := fakeProxy(t, map[string]string{
		"v1.2.5": "module example.com/vuln\n\ngo 1.20\n\nrequire example.com/other v1.0.0\n",
	})
	oldSum := "example.com/vuln v1.2.0 h1:old=\nexample.com/vuln v1.2.0/go.mod h1:oldmod=\n" +
		"example.com/other v1.0.0/go.mod h1:other=\n"

	res, err := BumpGoMod(context.Background(), proxy, []byte(projectMod), []byte(oldSum), testMod, "v1.2.5")
	require.NoError(t, err)

	require.Contains(t, string(res.GoMod), "example.com/vuln v1.2.5")
	require.Contains(t, string(res.GoMod), "example.com/other v1.0.0 // indirect")
	require.NotContains(t, string(res.GoMod), "v1.2.0")

	lines := strings.Split(strings.TrimSpace(string(res.GoSum)), "\n")
	require.Equal(t, []string{
		"example.com/other v1.0.0/go.mod h1:other=",
		"example.com/vuln v1.2.0 h1:old=",
		"example.com/vuln v1.2.0/go.mod h1:oldmod=",
		lines[3], // v1.2.5 zip
		lines[4], // v1.2.5 go.mod
	}, lines)
	require.True(t, strings.HasPrefix(lines[3], "example.com/vuln v1.2.5 h1:"), lines[3])
	require.True(t, strings.HasPrefix(lines[4], "example.com/vuln v1.2.5/go.mod h1:"), lines[4])

	// the hashes are deterministic: bumping again from the result is a no-op refusal, and the
	// sum computed for the same inputs is identical.
	res2, err := BumpGoMod(context.Background(), proxy, []byte(projectMod), []byte(oldSum), testMod, "v1.2.5")
	require.NoError(t, err)
	require.Equal(t, res.GoSum, res2.GoSum)
}

func TestBumpGoModRefusals(t *testing.T) {
	proxy := fakeProxy(t, map[string]string{
		"v1.2.5": "module example.com/vuln\n\ngo 1.20\n",
		"v1.3.0": "module example.com/vuln\n\ngo 1.23\n",
		"v1.4.0": "module example.com/vuln\n\ngo 1.20\n\nrequire example.com/new v1.0.0\n",
		"v1.5.0": "module example.com/vuln\n\ngo 1.20\n\nrequire example.com/other v1.1.0\n",
	})
	ctx := context.Background()

	for _, v := range []string{"v1.3.0", "v1.4.0", "v1.5.0"} {
		_, err := BumpGoMod(ctx, proxy, []byte(projectMod), nil, testMod, v)
		require.True(t, IsNeedsAgent(err), "%s: %v", v, err)
	}

	// other major line
	_, err := BumpGoMod(ctx, proxy, []byte(projectMod), nil, testMod, "v2.0.0")
	require.True(t, IsNeedsAgent(err), err)

	// replaced module
	replaced := projectMod + "\nreplace example.com/vuln => ../vuln\n"
	_, err = BumpGoMod(ctx, proxy, []byte(replaced), nil, testMod, "v1.2.5")
	require.True(t, IsNeedsAgent(err), err)

	// not required / already fixed
	_, err = BumpGoMod(ctx, proxy, []byte(projectMod), nil, "example.com/absent", "v1.0.1")
	require.ErrorIs(t, err, ErrUnsupported)
	_, err = BumpGoMod(ctx, proxy, []byte(projectMod), nil, testMod, "v1.1.0")
	require.ErrorIs(t, err, ErrUnsupported)
}
