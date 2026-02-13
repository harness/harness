//  Copyright 2023 Harness, Inc.
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

package pypi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/metadata/python"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/commons/pypi"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

var _ registry.PythonRegistry = (*adapter)(nil)
var _ adp.Adapter = (*adapter)(nil)

const (
	PyPiURL = "https://pypi.org"
)

type adapter struct {
	*native.Adapter
	registry types.UpstreamProxy
	client   *client
}

func newAdapter(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	registry types.UpstreamProxy,
	service secret.Service,
) (adp.Adapter, error) {
	nativeAdapter, err := native.NewAdapter(ctx, spaceFinder, service, registry)
	if err != nil {
		return nil, err
	}
	c, err := newClient(ctx, registry, spaceFinder, service)
	if err != nil {
		return nil, err
	}

	return &adapter{
		Adapter:  nativeAdapter,
		registry: registry,
		client:   c,
	}, nil
}

type factory struct {
}

func (f *factory) Create(
	ctx context.Context, spaceFinder refcache.SpaceFinder, record types.UpstreamProxy, service secret.Service,
) (adp.Adapter, error) {
	return newAdapter(ctx, spaceFinder, record, service)
}

func init() {
	adapterType := string(artifact.PackageTypePYTHON)
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Failed to register adapter factory for %s", adapterType)
		return
	}
}

func (a *adapter) GetMetadata(ctx context.Context, pkg string) (*pypi.SimpleMetadata, error) {
	basePath := "simple"
	if a.registry.Config != nil && a.registry.Config.RemoteUrlSuffix != "" {
		basePath = strings.Trim(a.registry.Config.RemoteUrlSuffix, "/")
	}

	filePath := path.Join(basePath, pkg) + "/"

	_, readCloser, err := a.GetFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer readCloser.Close()
	response, err := ParsePyPISimple(readCloser, a.GetURL(ctx, filePath))
	if err != nil {
		return nil, err
	}
	err = validateMetadata(response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func validateMetadata(response pypi.SimpleMetadata) error {
	for _, p := range response.Packages {
		if !p.Valid() {
			log.Error().Msgf("invalid package: %s", p.String())
			return fmt.Errorf("invalid package: %s", p.String())
		}
	}
	return nil
}

func (a *adapter) GetPackage(ctx context.Context, pkg string, filename string) (io.ReadCloser, error) {
	metadata, err := a.GetMetadata(ctx, pkg)
	if err != nil {
		return nil, err
	}

	downloadURL := ""

	for _, p := range metadata.Packages {
		if p.Name == filename {
			downloadURL = p.URL()
			break
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("pkg: %s, filename: %s not found", pkg, filename)
	}

	log.Ctx(ctx).Info().Msgf("Download URL: %s", downloadURL)
	_, closer, err := a.GetFileFromURL(ctx, downloadURL)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file from URL: %s", downloadURL)
		return nil, err
	}
	return closer, nil
}

func (a *adapter) GetJSON(ctx context.Context, pkg string, version string) (*python.Metadata, error) {
	_, readCloser, err := a.GetFile(ctx, fmt.Sprintf("pypi/%s/%s/json", pkg, version))
	if err != nil {
		return nil, err
	}
	defer readCloser.Close()
	response, err := ParseMetadata(ctx, readCloser)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func ParseMetadata(ctx context.Context, body io.ReadCloser) (python.Metadata, error) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return python.Metadata{}, err
	}

	var response Response
	if err := json.Unmarshal(bytes, &response); err != nil {
		// FIXME: This is known problem where if the response fields returns null, the null is not handled.
		// For eg: {"keywords":null} is not handled where "keywords" is []string
		log.Ctx(ctx).Warn().Err(err).Msgf("Failed to unmarshal response")
	}

	return response.Info, nil
}

// ParsePyPISimple parses the given HTML and returns a SimpleMetadata DTO.
func ParsePyPISimple(r io.ReadCloser, url string) (pypi.SimpleMetadata, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return pypi.SimpleMetadata{}, err
	}

	var result pypi.SimpleMetadata
	var packages []pypi.Package

	// Recursive function to walk the HTML nodes
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "meta":
				// Check for meta tag name="pypi:repository-version"
				var metaName, metaContent string
				for _, attr := range n.Attr {
					switch attr.Key {
					case "name":
						metaName = attr.Val
					case "content":
						metaContent = attr.Val
					}
				}
				if metaName == "pypi:repository-version" {
					result.MetaName = metaName
					result.Content = metaContent
				}
			case "title":
				if n.FirstChild != nil {
					result.Title = n.FirstChild.Data
				}
			case "a":
				// Capture all attributes in a map
				aMap := make(map[string]string)
				for _, attr := range n.Attr {
					aMap[attr.Key] = attr.Val
				}
				linkText := ""
				if n.FirstChild != nil {
					linkText = n.FirstChild.Data
				}
				packages = append(packages, pypi.Package{
					SimpleURL: url,
					ATags:     aMap,
					Name:      linkText,
				})
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	result.Packages = packages
	return result, nil
}
