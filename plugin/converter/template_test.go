package converter

import (
	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/golang/mock/gomock"
	"io/ioutil"
	"testing"
)

func TestTemplatePluginConvert(t *testing.T) {
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

	templateArgs, err := ioutil.ReadFile("testdata/starlark.template.yml")
	if err != nil {
		t.Error(err)
		return
	}

	template := &core.Template{
		Name: "plugin.starlark",
		Data: beforeInput,
	}

	controller := gomock.NewController(t)
	defer controller.Finish()

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name).Return(template, nil)

	plugin := TemplateConverter(templates)
	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.yml",
		},
		Config: &core.Config{
			Data: string(templateArgs),
		},
	}

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

	plugin := TemplateConverter(nil)
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
	plugin := TemplateConverter(nil)
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
	controller := gomock.NewController(t)
	defer controller.Finish()

	template := &core.Template{
		Name: "plugin.starlark",
		Data: nil,
	}

	templates := mock.NewMockTemplateStore(controller)
	templates.EXPECT().FindName(gomock.Any(), template.Name).Return(nil, nil)

	templateArgs, err := ioutil.ReadFile("testdata/starlark.template.yml")
	if err != nil {
		t.Error(err)
		return
	}

	plugin := TemplateConverter(templates)
	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.yml",
		},
		Config: &core.Config{Data: string(templateArgs)},
	}

	config, err := plugin.Convert(noContext, req)
	if config != nil {
		t.Errorf("template converter: template name given not found")
	}
}
