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

package starlark

import (
	"strings"

	"github.com/harness/gitness/types"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func createArgs(
	repo *types.Repository,
	pipeline *types.Pipeline,
	execution *types.Execution,
	repoIsPublic bool,
) []starlark.Value {
	args := []starlark.Value{
		starlarkstruct.FromStringDict(
			starlark.String("context"),
			starlark.StringDict{
				"repo":  starlarkstruct.FromStringDict(starlark.String("repo"), fromRepo(repo, pipeline, repoIsPublic)),
				"build": starlarkstruct.FromStringDict(starlark.String("build"), fromBuild(execution)),
			},
		),
	}
	return args
}

func fromBuild(v *types.Execution) starlark.StringDict {
	return starlark.StringDict{
		"event":         starlark.String(v.Event),
		"action":        starlark.String(v.Action),
		"cron":          starlark.String(v.Cron),
		"link":          starlark.String(v.Link),
		"branch":        starlark.String(v.Target),
		"source":        starlark.String(v.Source),
		"before":        starlark.String(v.Before),
		"after":         starlark.String(v.After),
		"target":        starlark.String(v.Target),
		"ref":           starlark.String(v.Ref),
		"commit":        starlark.String(v.After),
		"title":         starlark.String(v.Title),
		"message":       starlark.String(v.Message),
		"source_repo":   starlark.String(v.Fork),
		"author_login":  starlark.String(v.Author),
		"author_name":   starlark.String(v.AuthorName),
		"author_email":  starlark.String(v.AuthorEmail),
		"author_avatar": starlark.String(v.AuthorAvatar),
		"sender":        starlark.String(v.Sender),
		"debug":         starlark.Bool(v.Debug),
		"params":        fromMap(v.Params),
	}
}

func fromRepo(v *types.Repository, p *types.Pipeline, publicRepo bool) starlark.StringDict {
	namespace := v.Path
	idx := strings.LastIndex(v.Path, "/")
	if idx != -1 {
		namespace = v.Path[:idx]
	}

	return starlark.StringDict{
		// TODO [CODE-1363]: remove after identifier migration?
		"uid":                  starlark.String(v.Identifier),
		"identifier":           starlark.String(v.Identifier),
		"name":                 starlark.String(v.Identifier),
		"namespace":            starlark.String(namespace),
		"slug":                 starlark.String(v.Path),
		"git_http_url":         starlark.String(v.GitURL),
		"git_ssh_url":          starlark.String(v.GitURL),
		"link":                 starlark.String(v.GitURL),
		"branch":               starlark.String(v.DefaultBranch),
		"config":               starlark.String(p.ConfigPath),
		"private":              !starlark.Bool(publicRepo),
		"visibility":           starlark.String("internal"),
		"active":               starlark.Bool(true),
		"trusted":              starlark.Bool(true),
		"protected":            starlark.Bool(false),
		"ignore_forks":         starlark.Bool(false),
		"ignore_pull_requests": starlark.Bool(false),
	}
}

func fromMap(m map[string]string) *starlark.Dict {
	dict := new(starlark.Dict)
	for k, v := range m {
		//nolint: errcheck
		dict.SetKey(
			starlark.String(k),
			starlark.String(v),
		)
	}
	return dict
}
