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

package jsonnet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/types"

	"github.com/google/go-jsonnet"
)

const repo = "repo."
const build = "build."
const param = "param."

var noContext = context.Background()

type importer struct {
	repo      *types.Repository
	execution *types.Execution

	// jsonnet does not cache file imports and may request
	// the same file multiple times. We cache the files to
	// duplicate API calls.
	cache map[string]jsonnet.Contents

	// limit the number of outbound requests. github limits
	// the number of api requests per hour, so we should
	// make sure that a single build does not abuse the api
	// by importing dozens of files.
	limit int

	// counts the number of outbound requests. if the count
	// exceeds the limit, the importer will return errors.
	count int

	fileService file.Service
}

func (i *importer) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	if i.cache == nil {
		i.cache = map[string]jsonnet.Contents{}
	}

	// the import is relative to the imported from path. the
	// imported path must resolve to a filepath relative to
	// the root of the repository.
	importedPath = path.Join(
		path.Dir(importedFrom),
		importedPath,
	)

	if strings.HasPrefix(importedFrom, "../") {
		err = fmt.Errorf("jsonnet: cannot resolve import: %s", importedPath)
		return contents, foundAt, err
	}

	// if the contents exist in the cache, return the
	// cached item.
	var ok bool
	if contents, ok = i.cache[importedPath]; ok {
		return contents, importedPath, nil
	}

	defer func() {
		i.count++
	}()

	// if the import limit is exceeded log an error message.
	if i.limit > 0 && i.count >= i.limit {
		return contents, foundAt, errors.New("jsonnet: import limit exceeded")
	}

	find, err := i.fileService.Get(noContext, i.repo, importedPath, i.execution.Ref)

	if err != nil {
		return contents, foundAt, err
	}

	i.cache[importedPath] = jsonnet.MakeContents(string(find.Data))

	return i.cache[importedPath], importedPath, err
}

func Parse(
	repo *types.Repository,
	repoIsPublic bool,
	pipeline *types.Pipeline,
	execution *types.Execution,
	file *file.File,
	fileService file.Service,
	limit int,
) (string, error) {
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500
	vm.StringOutput = false
	vm.ErrorFormatter.SetMaxStackTraceSize(20)
	if fileService != nil && limit > 0 {
		vm.Importer(
			&importer{
				repo:        repo,
				execution:   execution,
				limit:       limit,
				fileService: fileService,
			},
		)
	}

	// map execution/repo/pipeline parameters
	if execution != nil {
		mapBuild(execution, vm)
	}
	if repo != nil {
		mapRepo(repo, pipeline, vm, repoIsPublic)
	}

	jsonnetFile := file
	jsonnetFileName := pipeline.ConfigPath

	// convert the jsonnet file to yaml
	buf := new(bytes.Buffer)
	docs, err := vm.EvaluateAnonymousSnippetStream(jsonnetFileName, string(jsonnetFile.Data))
	if err != nil {
		doc, err2 := vm.EvaluateAnonymousSnippet(jsonnetFileName, string(jsonnetFile.Data))
		if err2 != nil {
			return "", err
		}
		docs = append(docs, doc)
	}

	// the jsonnet vm returns a stream of yaml documents
	// that need to be combined into a single yaml file.
	for _, doc := range docs {
		buf.WriteString("---")
		buf.WriteString("\n")
		buf.WriteString(doc)
	}

	return buf.String(), nil
}

// mapBuild populates build variables available to jsonnet templates.
// Since we want to maintain compatibility with drone, the older format
// needs to be maintained (even if the variables do not exist in gitness).
func mapBuild(v *types.Execution, vm *jsonnet.VM) {
	vm.ExtVar(build+"event", v.Event)
	vm.ExtVar(build+"action", v.Action)
	vm.ExtVar(build+"environment", v.Deploy)
	vm.ExtVar(build+"link", v.Link)
	vm.ExtVar(build+"branch", v.Target)
	vm.ExtVar(build+"source", v.Source)
	vm.ExtVar(build+"before", v.Before)
	vm.ExtVar(build+"after", v.After)
	vm.ExtVar(build+"target", v.Target)
	vm.ExtVar(build+"ref", v.Ref)
	vm.ExtVar(build+"commit", v.After)
	vm.ExtVar(build+"ref", v.Ref)
	vm.ExtVar(build+"title", v.Title)
	vm.ExtVar(build+"message", v.Message)
	vm.ExtVar(build+"source_repo", v.Fork)
	vm.ExtVar(build+"author_login", v.Author)
	vm.ExtVar(build+"author_name", v.AuthorName)
	vm.ExtVar(build+"author_email", v.AuthorEmail)
	vm.ExtVar(build+"author_avatar", v.AuthorAvatar)
	vm.ExtVar(build+"sender", v.Sender)
	fromMap(v.Params, vm)
}

// mapBuild populates repo level variables available to jsonnet templates.
// Since we want to maintain compatibility with drone 2.x, the older format
// needs to be maintained (even if the variables do not exist in gitness).
func mapRepo(v *types.Repository, p *types.Pipeline, vm *jsonnet.VM, publicRepo bool) {
	namespace := v.Path
	idx := strings.LastIndex(v.Path, "/")
	if idx != -1 {
		namespace = v.Path[:idx]
	}
	// TODO [CODE-1363]: remove after identifier migration.
	vm.ExtVar(repo+"uid", v.Identifier)
	vm.ExtVar(repo+"identifier", v.Identifier)
	vm.ExtVar(repo+"name", v.Identifier)
	vm.ExtVar(repo+"namespace", namespace)
	vm.ExtVar(repo+"slug", v.Path)
	vm.ExtVar(repo+"git_http_url", v.GitURL)
	vm.ExtVar(repo+"git_ssh_url", v.GitURL)
	vm.ExtVar(repo+"link", v.GitURL)
	vm.ExtVar(repo+"branch", v.DefaultBranch)
	vm.ExtVar(repo+"config", p.ConfigPath)
	vm.ExtVar(repo+"private", strconv.FormatBool(!publicRepo))
	vm.ExtVar(repo+"visibility", "internal")
	vm.ExtVar(repo+"active", strconv.FormatBool(true))
	vm.ExtVar(repo+"trusted", strconv.FormatBool(true))
	vm.ExtVar(repo+"protected", strconv.FormatBool(false))
	vm.ExtVar(repo+"ignore_forks", strconv.FormatBool(false))
	vm.ExtVar(repo+"ignore_pull_requests", strconv.FormatBool(false))
}

func fromMap(m map[string]string, vm *jsonnet.VM) {
	for k, v := range m {
		vm.ExtVar(build+param+k, v)
	}
}
