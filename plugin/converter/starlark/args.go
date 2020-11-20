// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package starlark

import (
	"github.com/drone/drone/core"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// TODO(bradrydzewski) add repository id
// TODO(bradrydzewski) add repository timeout
// TODO(bradrydzewski) add repository counter
// TODO(bradrydzewski) add repository created
// TODO(bradrydzewski) add repository updated
// TODO(bradrydzewski) add repository synced
// TODO(bradrydzewski) add repository version

// TODO(bradrydzewski) add build id, will always be zero value
// TODO(bradrydzewski) add build number, will always be zero value
// TODO(bradrydzewski) add build started, will always be zero value
// TODO(bradrydzewski) add build finished, will always be zero value
// TODO(bradrydzewski) add build created, will always be zero value
// TODO(bradrydzewski) add build updated, will always be zero value
// TODO(bradrydzewski) add build parent
// TODO(bradrydzewski) add build timestamp

func createArgs(repo *core.Repository, build *core.Build) []starlark.Value {
	return []starlark.Value{
		starlarkstruct.FromStringDict(
			starlark.String("context"),
			starlark.StringDict{
				"repo":  starlarkstruct.FromStringDict(starlark.String("repo"), fromRepo(repo)),
				"build": starlarkstruct.FromStringDict(starlark.String("build"), fromBuild(build)),
			},
		),
	}
}

func fromBuild(v *core.Build) starlark.StringDict {
	return starlark.StringDict{
		"event":         starlark.String(v.Event),
		"action":        starlark.String(v.Action),
		"cron":          starlark.String(v.Cron),
		"environment":   starlark.String(v.Deploy),
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
		"params":        fromMap(v.Params),
	}
}

func fromRepo(v *core.Repository) starlark.StringDict {
	return starlark.StringDict{
		"uid":                  starlark.String(v.UID),
		"name":                 starlark.String(v.Name),
		"namespace":            starlark.String(v.Namespace),
		"slug":                 starlark.String(v.Slug),
		"git_http_url":         starlark.String(v.HTTPURL),
		"git_ssh_url":          starlark.String(v.SSHURL),
		"link":                 starlark.String(v.Link),
		"branch":               starlark.String(v.Branch),
		"config":               starlark.String(v.Config),
		"private":              starlark.Bool(v.Private),
		"visibility":           starlark.String(v.Visibility),
		"active":               starlark.Bool(v.Active),
		"trusted":              starlark.Bool(v.Trusted),
		"protected":            starlark.Bool(v.Protected),
		"ignore_forks":         starlark.Bool(v.IgnoreForks),
		"ignore_pull_requests": starlark.Bool(v.IgnorePulls),
	}
}

func fromMap(m map[string]string) *starlark.Dict {
	dict := new(starlark.Dict)
	for k, v := range m {
		dict.SetKey(
			starlark.String(k),
			starlark.String(v),
		)
	}
	return dict
}
