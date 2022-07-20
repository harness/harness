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
	"encoding/json"
	"errors"
	"io/ioutil"
	"runtime"
	"strings"
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

	plugin := Template(templates, 0)
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

	plugin := Template(nil, 0)
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
	plugin := Template(nil, 0)
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

// Test makes sure that we don't skip templating for neither the "yml" or "yaml" extension.
func TestTemplatePluginConvertDroneFileYamlExtensions(t *testing.T) {
	extensions := []string{"yml", "yaml"}
	dummyErr := errors.New("dummy-error")

	for _, extension := range extensions {
		t.Run(extension, func(t *testing.T) {
			args, err := ioutil.ReadFile("testdata/yaml.template.yml")
			if err != nil {
				t.Error(err)
				return
			}

			controller := gomock.NewController(t)
			defer controller.Finish()

			templates := mock.NewMockTemplateStore(controller)
			templates.EXPECT().FindName(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dummyErr)

			plugin := Template(templates, 0)
			req := &core.ConvertArgs{
				Build: &core.Build{
					After: "3d21ec53a331a6f037a91c368710b99387d012c1",
				},
				Repo: &core.Repository{
					Slug:   "octocat/hello-world",
					Config: ".drone." + extension,
				},
				Config: &core.Config{Data: string(args)},
			}

			_, err = plugin.Convert(noContext, req)
			if err != nil && err != dummyErr {
				t.Error(err)
			}
			if err == nil {
				t.Errorf("Templating was skipped")
			}
		})
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

	plugin := Template(templates, 0)

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

	plugin := Template(templates, 0)
	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}

	if config == nil {
		t.Error("Want non-nil configuration")
		return
	}

	want := string(after)
	// on windows line endings are \r\n, lets change them to linux for comparison
	if runtime.GOOS == "windows" {
		want = strings.Replace(want, "\r\n", "\n", -1)
	}

	got := config.Data
	if want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

func TestTemplateNestedValuesPluginConvertStarlark(t *testing.T) {
	type Pipeline struct {
		Kind  string `json:"kind"`
		Name  string `json:"name"`
		Steps []struct {
			Name     string   `json:"name"`
			Image    string   `json:"image"`
			Commands []string `json:"commands"`
		} `json:"steps"`
	}

	templateArgs, err := ioutil.ReadFile("testdata/starlark-nested.template.yml")
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

	beforeInput, err := ioutil.ReadFile("testdata/starlark.input-nested.star")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("testdata/starlark.input-nested.star.golden")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name:      "test.nested.starlark",
		Data:      string(beforeInput),
		Namespace: "octocat",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(template, nil)

	plugin := Template(templates, 0)
	config, err := plugin.Convert(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}

	if config == nil {
		t.Error("Want non-nil configuration")
		return
	}
	result := Pipeline{}
	err = json.Unmarshal(after, &result)
	beforeConfig := Pipeline{}
	err = json.Unmarshal([]byte(config.Data), &beforeConfig)

	if want, got := beforeConfig.Name, result.Name; want != got {
		t.Errorf("Want %q got %q", want, got)
	}
	if want, got := beforeConfig.Kind, result.Kind; want != got {
		t.Errorf("Want %q got %q", want, got)
	}
	if want, got := beforeConfig.Steps[0].Name, result.Steps[0].Name; want != got {
		t.Errorf("Want %q got %q", want, got)
	}
	if want, got := beforeConfig.Steps[0].Commands[0], result.Steps[0].Commands[0]; want != got {
		t.Errorf("Want %q got %q", want, got)
	}
	if want, got := beforeConfig.Steps[0].Image, result.Steps[0].Image; want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

func TestTemplatePluginConvertYaml(t *testing.T) {
	templateArgs, err := ioutil.ReadFile("testdata/yaml.template.yml")
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

	beforeInput, err := ioutil.ReadFile("testdata/yaml.input.yml")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("testdata/yaml.input.golden")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name:      "plugin.yaml",
		Data:      string(beforeInput),
		Namespace: "octocat",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(template, nil)

	plugin := Template(templates, 0)
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

// tests to check error is thrown if user has already loaded a template file of invalid extension
// and refers to it in the drone.yml file
func TestTemplatePluginConvertInvalidTemplateExtension(t *testing.T) {
	// reads yml input file which refers to a template file of invalid extensions
	templateArgs, err := ioutil.ReadFile("testdata/yaml.template.invalid.yml")
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
	// reads the template drone.yml
	beforeInput, err := ioutil.ReadFile("testdata/yaml.input.yml")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name:      "plugin.txt",
		Data:      string(beforeInput),
		Namespace: "octocat",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(template, nil)

	plugin := Template(templates, 0)
	config, err := plugin.Convert(noContext, req)
	if config != nil {
		t.Errorf("template extension invalid. must be yaml, starlark or jsonnet")
	}
}

func TestTemplatePluginConvertYamlWithComment(t *testing.T) {
	templateArgs, err := ioutil.ReadFile("testdata/yaml.template.comment.yml")
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

	beforeInput, err := ioutil.ReadFile("testdata/yaml.input.yml")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("testdata/yaml.input.golden")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name:      "plugin.yaml",
		Data:      string(beforeInput),
		Namespace: "octocat",
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name, req.Repo.Namespace).Return(template, nil)

	plugin := Template(templates, 0)
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
