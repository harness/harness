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

package swagger

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/openapi-go/openapi3"
	"gopkg.in/alecthomas/kingpin.v2"
)

type infOpenAPI struct{}

func (infOpenAPI) Generate() *openapi3.Spec {
	spec := &openapi3.Spec{Openapi: "3.0.3"}
	spec.Info.Title = "Inf API"
	spec.Info.Version = "1.0.0"
	return spec
}

func TestInfSwaggerCommand_WritesToFile(t *testing.T) {
	app := kingpin.New("test", "")
	app.Terminate(nil)
	Register(app, infOpenAPI{})

	path := filepath.Join(t.TempDir(), "swagger.yaml")
	_, err := app.Parse([]string{"swagger", path})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "title: Inf API")
	require.Contains(t, string(data), "openapi: 3.0.3")

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestInfSwaggerCommand_WritesToStdout(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	app := kingpin.New("test", "")
	app.Terminate(nil)
	Register(app, infOpenAPI{})
	_, err = app.Parse([]string{"swagger"})
	require.NoError(t, err)
	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Contains(t, string(out), "title: Inf API")
}

func TestInfSwaggerCommand_UnwritablePath(t *testing.T) {
	app := kingpin.New("test", "")
	app.Terminate(nil)
	Register(app, infOpenAPI{})
	_, err := app.Parse([]string{"swagger", filepath.Join(t.TempDir(), "missing", "swagger.yaml")})
	require.Error(t, err)
}
