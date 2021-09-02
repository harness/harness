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

package converter

import (
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
)

func TestStarlarkConvert(t *testing.T) {
	plugin := Starlark(true, 0)

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

	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}
	if config != nil {
		t.Error("Want nil config when configuration is not starlark file")
		return
	}

	before, err := ioutil.ReadFile("testdata/single.star")
	if err != nil {
		t.Error(err)
		return
	}
	after, err := ioutil.ReadFile("testdata/single.star.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req.Repo.Config = "single.star"
	req.Config.Data = string(before)
	config, err = plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}
	if config == nil {
		t.Error("Want non-nil configuration")
		return
	}

	if want, got := config.Data, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

// this test verifies the starlark file can generate a multi-document
// yaml file that defines multiple pipelines.
func TestConvert_Multi(t *testing.T) {
	before, err := ioutil.ReadFile("testdata/multi.star")
	if err != nil {
		t.Error(err)
		return
	}
	after, err := ioutil.ReadFile("testdata/multi.star.golden")
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
		Config: &core.Config{
			Data: string(before),
		},
	}

	plugin := Starlark(true, 0)
	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}

	config, err = plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}
	if config == nil {
		t.Error("Want non-nil configuration")
		return
	}

	if want, got := config.Data, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

// this test verifies the plugin is skipped when it has
// not been explicitly enabled.
func TestConvert_Skip(t *testing.T) {
	plugin := Starlark(false, 0)
	config, err := plugin.Convert(noContext, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if config != nil {
		t.Errorf("Expect nil config returned when plugin disabled")
	}
}

// this test verifies the plugin is skipped when the config
// file extension is not a starlark extension.
func TestConvert_SkipYaml(t *testing.T) {
	req := &core.ConvertArgs{
		Repo: &core.Repository{
			Config: ".drone.yaml",
		},
	}

	plugin := Starlark(true, 0)
	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}
	if config != nil {
		t.Errorf("Expect nil config returned for non-starlark files")
	}
}
