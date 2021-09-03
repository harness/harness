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
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
)

func TestParseStarlark(t *testing.T) {
	before, err := ioutil.ReadFile("../testdata/starlark.input.star")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("../testdata/starlark.input.star.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.yml",
		},
		Config: &core.Config{},
	}
	template := &core.Template{
		Name: "my_template.star",
		Data: string(before),
	}

	templateData := map[string]interface{}{
		"stepName": "my_step",
		"image":    "my_image",
		"commands": "my_command",
	}

	req.Config.Data = string(before)

	parsedFile, err := Parse(req, template, templateData, 0)
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := parsedFile, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

func TestParseStarlarkNotTemplateFile(t *testing.T) {
	before, err := ioutil.ReadFile("../testdata/single.star")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("../testdata/single.star.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.star",
		},
		Config: &core.Config{},
	}

	req.Repo.Config = "plugin.starlark.star"
	req.Config.Data = string(before)

	parsedFile, err := Parse(req, nil, nil, 0)
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := parsedFile, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}
