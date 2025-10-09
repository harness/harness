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

package resolver

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	v1yaml "github.com/drone/spec/dist/go"
	"github.com/drone/spec/dist/go/parse"
	"github.com/rs/zerolog/log"
)

// Lookup returns a resource by name, kind and type. It also sends in the
// execution ID.
type LookupFunc func(name, kind, typ, version string, id int64) (*v1yaml.Config, error)

type Manager struct {
	config         *types.Config
	pluginStore    store.PluginStore
	templateStore  store.TemplateStore
	executionStore store.ExecutionStore
	repoStore      store.RepoStore
}

func NewManager(
	config *types.Config,
	pluginStore store.PluginStore,
	templateStore store.TemplateStore,
	executionStore store.ExecutionStore,
	repoStore store.RepoStore,
) *Manager {
	return &Manager{
		config:         config,
		pluginStore:    pluginStore,
		templateStore:  templateStore,
		executionStore: executionStore,
		repoStore:      repoStore,
	}
}

// GetLookupFn returns a lookup function for plugins and templates which can be used in the resolver
// passed to the drone runner.
//
//nolint:gocognit
func (m *Manager) GetLookupFn() LookupFunc {
	noContext := context.Background()
	return func(name, kind, typ, version string, executionID int64) (*v1yaml.Config, error) {
		// Find space ID corresponding to the executionID
		execution, err := m.executionStore.Find(noContext, executionID)
		if err != nil {
			return nil, fmt.Errorf("could not find relevant execution: %w", err)
		}

		// Find the repo so we know in which space templates should be searched
		repo, err := m.repoStore.Find(noContext, execution.RepoID)
		if err != nil {
			return nil, fmt.Errorf("could not find relevant repo: %w", err)
		}

		f := Resolve(noContext, m.pluginStore, m.templateStore, repo.ParentID)
		return f(name, kind, typ, version)
	}
}

// Populate fetches plugins information from an external source or a local zip
// and populates in the DB.
func (m *Manager) Populate(ctx context.Context) error {
	pluginsURL := m.config.CI.PluginsZipURL
	if pluginsURL == "" {
		return fmt.Errorf("plugins url not provided to read schemas from")
	}

	var zipFile *zip.ReadCloser
	if _, err := os.Stat(pluginsURL); err != nil { // local path doesn't exist - must be a remote link
		// Download zip file locally
		f, err := os.CreateTemp(os.TempDir(), "plugins.zip")
		if err != nil {
			return fmt.Errorf("could not create temp file: %w", err)
		}
		defer os.Remove(f.Name())
		err = downloadZip(ctx, pluginsURL, f.Name())
		if err != nil {
			return fmt.Errorf("could not download remote zip: %w", err)
		}
		pluginsURL = f.Name()
	}
	// open up a zip reader for the file
	zipFile, err := zip.OpenReader(pluginsURL)
	if err != nil {
		return fmt.Errorf("could not open zip for reading: %w", err)
	}
	defer zipFile.Close()

	// upsert any new plugins.
	err = m.traverseAndUpsertPlugins(ctx, zipFile)
	if err != nil {
		return fmt.Errorf("could not upsert plugins: %w", err)
	}

	return nil
}

// downloadZip is a helper function that downloads a zip from a URL and
// writes it to a path in the local filesystem.
//
//nolint:gosec // URL is coming from environment variable (user configured it)
func downloadZip(ctx context.Context, pluginURL, path string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pluginURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("could not get zip from url: %w", err)
	}
	// ensure the body is closed after we read (independent of status code or error)
	if response != nil && response.Body != nil {
		// Use function to satisfy the linter which complains about unhandled errors otherwise
		defer func() { _ = response.Body.Close() }()
	}

	// Create the file on the local FS. If it exists, it will be truncated.
	output, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
	}
	defer output.Close()

	// Copy the zip output to the file.
	_, err = io.Copy(output, response.Body)
	if err != nil {
		return fmt.Errorf("could not copy response body output to file: %w", err)
	}

	return nil
}

// traverseAndUpsertPlugins traverses through the zip and upserts plugins into the database
// if they are not present.
//
//nolint:gocognit // refactor if needed.
func (m *Manager) traverseAndUpsertPlugins(ctx context.Context, rc *zip.ReadCloser) error {
	plugins, err := m.pluginStore.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("could not list plugins: %w", err)
	}
	// Put the plugins in a map so we don't have to perform frequent DB queries.
	pluginMap := map[string]*types.Plugin{}
	for _, p := range plugins {
		pluginMap[p.Identifier] = p
	}
	cnt := 0
	for _, file := range rc.File {
		matched, err := filepath.Match("**/plugins/*/*.yaml", file.Name)
		if err != nil { // only returns BadPattern error which shouldn't happen
			return fmt.Errorf("could not glob pattern: %w", err)
		}
		if !matched {
			continue
		}
		fc, err := file.Open()
		if err != nil {
			log.Warn().Err(err).Str("name", file.Name).Msg("could not open file")
			continue
		}
		defer fc.Close()
		var buf bytes.Buffer
		_, err = io.Copy(&buf, fc) //nolint:gosec // plugin source is configured via environment variables by user
		if err != nil {
			log.Warn().Err(err).Str("name", file.Name).Msg("could not read file contents")
			continue
		}
		// schema should be a valid config - if not log an error and continue.
		config, err := parse.ParseBytes(buf.Bytes())
		if err != nil {
			log.Warn().Err(err).Str("name", file.Name).Msg("could not parse schema into valid config")
			continue
		}

		var desc string
		switch vv := config.Spec.(type) {
		case *v1yaml.PluginStep:
			desc = vv.Description
		case *v1yaml.PluginStage:
			desc = vv.Description
		default:
			log.Warn().Str("name", file.Name).Msg("schema did not match a valid plugin schema")
			continue
		}

		plugin := &types.Plugin{
			Description: desc,
			Identifier:  config.Name,
			Type:        config.Type,
			Spec:        buf.String(),
		}

		// Try to read the logo if it exists in the same directory
		dir := filepath.Dir(file.Name)
		logoFile := filepath.Join(dir, "logo.svg")
		if lf, err := rc.Open(logoFile); err == nil { // if we can open the logo file
			var lbuf bytes.Buffer
			_, err = io.Copy(&lbuf, lf)
			if err != nil {
				log.Warn().Err(err).Str("name", file.Name).Msg("could not copy logo file")
			} else {
				plugin.Logo = lbuf.String()
			}
		}

		// If plugin already exists in the database, skip upsert
		if p, ok := pluginMap[plugin.Identifier]; ok {
			if p.Matches(plugin) {
				continue
			}
		}

		// If plugin name exists with a different spec, call update - otherwise call create.
		// TODO: Once we start using versions, we can think of whether we want to
		// keep different schemas for each version in the database. For now, we will
		// simply overwrite the existing version with the new version.
		if _, ok := pluginMap[plugin.Identifier]; ok {
			err = m.pluginStore.Update(ctx, plugin)
			if err != nil {
				log.Warn().Str("name", file.Name).Err(err).Msg("could not update plugin")
				continue
			}
			log.Info().Str("name", file.Name).Msg("detected changes: updated existing plugin entry")
		} else {
			err = m.pluginStore.Create(ctx, plugin)
			if err != nil {
				log.Warn().Str("name", file.Name).Err(err).Msg("could not create plugin in DB")
				continue
			}
			cnt++
		}
	}
	log.Info().Msgf("added %d new entries to plugins", cnt)
	return nil
}
