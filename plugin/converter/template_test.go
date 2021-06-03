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
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
)

func TestTemplatePluginConvertStarlark(t *testing.T) {
	templateArgs, err := ioutil.ReadFile("testdata/starlark.template.yml")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:      "octocat/hello-world",
			Config:    ".drone.yml",
			Namespace: "octocat",
		},
		Config: &core.Config{
			Data: string(templateArgs),
		},
	}

	beforeInput, err := ioutil.ReadFile("testdata/starlark.input.star")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("testdata/starlark.input.star.golden")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name:      "plugin.starlark",
		Data:      string(beforeInput),
		Namespace: "octocat",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(template, nil)

	plugin := Template(templates)
	config, err := plugin.Convert(noContext, req)
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

func TestTemplatePluginConvertNotYamlFile(t *testing.T) {

	plugin := Template(nil)
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

	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}
	if config != nil {
		t.Errorf("Expect nil config returned for non-starlark files")
	}
}

func TestTemplatePluginConvertDroneFileTypePipeline(t *testing.T) {
	args, err := ioutil.ReadFile("testdata/drone.yml")
	if err != nil {
		t.Error(err)
		return
	}
	plugin := Template(nil)
	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.yml",
		},
		Config: &core.Config{Data: string(args)},
	}

	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}
	if config != nil {
		t.Errorf("Expect nil config returned for non-starlark files")
	}
}

func TestTemplatePluginConvertTemplateNotFound(t *testing.T) {
	templateArgs, err := ioutil.ReadFile("testdata/starlark.template.yml")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:      "octocat/hello-world",
			Config:    ".drone.yml",
			Namespace: "octocat",
		},
		Config: &core.Config{Data: string(templateArgs)},
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	template := &core.Template{
		Name: "plugin.starlark",
		Data: "",
	}

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(nil, nil)

	plugin := Template(templates)

	config, err := plugin.Convert(noContext, req)
	if config != nil {
		t.Errorf("template converter: template name given not found")
	}
}

func TestTemplatePluginConvertJsonnet(t *testing.T) {
	templateArgs, err := ioutil.ReadFile("testdata/jsonnet.template.yml")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:      "octocat/hello-world",
			Config:    ".drone.yml",
			Namespace: "octocat",
		},
		Config: &core.Config{
			Data: string(templateArgs),
		},
	}

	beforeInput, err := ioutil.ReadFile("testdata/input.jsonnet")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("testdata/input.jsonnet.golden")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name:      "plugin.jsonnet",
		Data:      string(beforeInput),
		Namespace: "octocat",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(template, nil)

	plugin := Template(templates)
	config, err := plugin.Convert(noContext, req)
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
