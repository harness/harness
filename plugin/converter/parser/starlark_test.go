package parser

import (
	"github.com/drone/drone/core"
	"io/ioutil"
	"testing"
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
		Data: before,
	}

	templateData := map[string]interface{}{
		"stepName": "my_step",
		"image":    "my_image",
		"commands": "my_command",
	}

	req.Config.Data = string(before)

	parsedFile, err := ParseStarlark(req, template, templateData)
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := *parsedFile, string(after); want != got {
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

	parsedFile, err := ParseStarlark(req, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := *parsedFile, string(after); want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}
