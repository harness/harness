// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package converter

import (
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
)

const jsonnetFile = `{"foo": "bar"}`
const jsonnetFileAfter = `---
{
   "foo": "bar"
}
`

const jsonnetStream = `[{"foo": "bar"}]`
const jsonnetStreamAfter = `---
{
   "foo": "bar"
}
`

const jsonnetFileImport = `local step = import '.step.libsonnet';
{"foo": ["bar"], "steps": [step]}`
const jsonnetFileImportLib = `{"image": "app"}`
const jsonnetFileImportAfter = `---
{
   "foo": [
      "bar"
   ],
   "steps": [
      {
         "image": "app"
      }
   ]
}
`

const jsonnetFileMultipleImports = `local step = import '.step.libsonnet';
local step2 = import '.step2.jsonnet';
{"foo": ["bar"], "steps": [step, step2]}`

func TestJsonnet_Stream(t *testing.T) {
	args := &core.ConvertArgs{
		Repo:   &core.Repository{Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetStream},
	}
	service := Jsonnet(true, 0, nil)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil {
		t.Errorf("Expected a converted file, got nil")
		return
	}
	if got, want := res.Data, jsonnetStreamAfter; got != want {
		t.Errorf("Want converted file %q, got %q", want, got)
	}
}

func TestJsonnet_Snippet(t *testing.T) {
	args := &core.ConvertArgs{
		Repo:   &core.Repository{Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFile},
	}
	service := Jsonnet(true, 0, nil)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
		return
	}
	if res == nil {
		t.Errorf("Expected a converted file, got nil")
		return
	}
	if got, want := res.Data, jsonnetFileAfter; got != want {
		t.Errorf("Want converted file %q, got %q", want, got)
	}
}

func TestJsonnet_Error(t *testing.T) {
	args := &core.ConvertArgs{
		Repo:   &core.Repository{Config: ".drone.jsonnet"},
		Config: &core.Config{Data: "\\"}, // invalid jsonnet
	}
	service := Jsonnet(true, 0, nil)
	_, err := service.Convert(noContext, args)
	if err == nil {
		t.Errorf("Expect jsonnet parsing error, got nil")
	}
}

func TestJsonnet_Disabled(t *testing.T) {
	service := Jsonnet(false, 0, nil)
	res, err := service.Convert(noContext, nil)
	if err != nil {
		t.Error(err)
	}
	if res != nil {
		t.Errorf("Expect nil response when disabled")
	}
}

func TestJsonnet_NotJsonnet(t *testing.T) {
	args := &core.ConvertArgs{
		Repo: &core.Repository{Config: ".drone.yml"},
	}
	service := Jsonnet(true, 0, nil)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
	}
	if res != nil {
		t.Errorf("Expect nil response when not jsonnet")
	}
}

func TestJsonnet_Import(t *testing.T) {
	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref:   "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
			After: "542ed565d03dab86f079798f937663ec1f05360b",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFileImport},
		User: &core.User{
			Token: "foobar",
		},
	}
	importedContent := &core.File{
		Data: []byte(jsonnetFileImportLib),
	}
	controller := gomock.NewController(t)
	mockFileService := mock.NewMockFileService(controller)
	mockFileService.EXPECT().Find(gomock.Any(), &core.User{Token: "foobar"}, "octocat/hello-world", "542ed565d03dab86f079798f937663ec1f05360b", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", ".step.libsonnet").Return(importedContent, nil).Times(2)
	service := Jsonnet(true, 1, mockFileService)
	res, err := service.Convert(noContext, args)
	if err != nil {
		t.Error(err)
	}
	if got, want := res.Data, jsonnetFileImportAfter; got != want {
		t.Errorf("Want converted file:\n%q\ngot\n%q", want, got)
	}
}

func TestJsonnet_ImportLimit(t *testing.T) {
	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref:   "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
			After: "542ed565d03dab86f079798f937663ec1f05360b",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFileMultipleImports},
		User: &core.User{
			Token: "foobar",
		},
	}
	importedContent := &core.File{
		Data: []byte(jsonnetFileImportLib),
	}
	controller := gomock.NewController(t)
	mockFileService := mock.NewMockFileService(controller)
	mockFileService.EXPECT().Find(gomock.Any(), &core.User{Token: "foobar"}, "octocat/hello-world", "542ed565d03dab86f079798f937663ec1f05360b", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", ".step.libsonnet").Return(importedContent, nil).Times(2)

	service := Jsonnet(true, 1, mockFileService)
	_, err := service.Convert(noContext, args)
	if err == nil {
		t.Errorf("Expect nil response when jsonnet import limit is exceeded")
	}
}

func TestJsonnet_LimitZero(t *testing.T) {
	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref:   "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
			After: "542ed565d03dab86f079798f937663ec1f05360b",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFile},
		User: &core.User{
			Token: "foobar",
		},
	}

	controller := gomock.NewController(t)
	mockFileService := mock.NewMockFileService(controller)
	mockFileService.EXPECT().Find(gomock.Any(), &core.User{Token: "foobar"}, "octocat/hello-world", "542ed565d03dab86f079798f937663ec1f05360b", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", ".step.libsonnet").Times(0)

	service := Jsonnet(true, 0, mockFileService)
	res, err := service.Convert(noContext, args)

	if err != nil {
		t.Error(err)
		return
	}
	if got, want := res.Data, jsonnetFileAfter; got != want {
		t.Errorf("Want converted file %q, got %q", want, got)
	}
}

func TestJsonnet_ImportLimitZero(t *testing.T) {
	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref:   "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
			After: "542ed565d03dab86f079798f937663ec1f05360b",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFileImport},
		User: &core.User{
			Token: "foobar",
		},
	}
	importedContent := &core.File{
		Data: []byte(jsonnetFileImportLib),
	}
	controller := gomock.NewController(t)
	mockFileService := mock.NewMockFileService(controller)
	mockFileService.EXPECT().Find(gomock.Any(), &core.User{Token: "foobar"}, "octocat/hello-world", "542ed565d03dab86f079798f937663ec1f05360b", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", ".step.libsonnet").Return(importedContent, nil).Times(2)

	service := Jsonnet(true, 0, mockFileService)
	_, err := service.Convert(noContext, args)
	if err == nil {
		t.Errorf("Expect nil response when jsonnet import limit is exceeded")
	}
}

func TestJsonnet_ImportFileServiceNil(t *testing.T) {
	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref:   "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
			After: "542ed565d03dab86f079798f937663ec1f05360b",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFileMultipleImports},
		User: &core.User{
			Token: "foobar",
		},
	}

	service := Jsonnet(true, 1, nil)
	_, err := service.Convert(noContext, args)
	if err == nil {
		t.Errorf("Expect nil response when jsonnet import limit is exceeded")
	}
}

func TestJsonnet_FileServiceNil(t *testing.T) {
	args := &core.ConvertArgs{
		Build: &core.Build{
			Ref:   "a6586b3db244fb6b1198f2b25c213ded5b44f9fa",
			After: "542ed565d03dab86f079798f937663ec1f05360b",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet"},
		Config: &core.Config{Data: jsonnetFile},
		User: &core.User{
			Token: "foobar",
		},
	}

	service := Jsonnet(true, 1, nil)
	res, err := service.Convert(noContext, args)

	if err != nil {
		t.Error(err)
		return
	}
	if got, want := res.Data, jsonnetFileAfter; got != want {
		t.Errorf("Want converted file %q, got %q", want, got)
	}
}