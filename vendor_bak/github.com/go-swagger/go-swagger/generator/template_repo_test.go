package generator

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	singleTemplate      = `test`
	multipleDefinitions = `{{ define "T1" }}T1{{end}}{{ define "T2" }}T2{{end}}`
	dependantTemplate   = `{{ template "T1" }}D1`
	cirularDeps1        = `{{ define "T1" }}{{ .Name }}: {{ range .Children }}{{ template "T2" . }}{{end}}{{end}}{{template "T1" . }}`
	cirularDeps2        = `{{ define "T2" }}{{if .Recurse }}{{ template "T1" . }}{{ else }}Children{{end}}{{end}}`
)

func TestRepoLoadingTemplates(t *testing.T) {

	repo := NewRepository(nil)

	repo.AddFile("simple", singleTemplate)

	templ, err := repo.Get("simple")

	assert.Nil(t, err)

	var b bytes.Buffer

	err = templ.Execute(&b, nil)

	assert.Nil(t, err)

	assert.Equal(t, "test", b.String())
}

func TestRepoLoadsAllTemplatesDefined(t *testing.T) {

	var b bytes.Buffer
	repo := NewRepository(nil)

	repo.AddFile("multiple", multipleDefinitions)

	templ, err := repo.Get("multiple")
	assert.Nil(t, err)
	err = templ.Execute(&b, nil)
	assert.Nil(t, err)

	assert.Equal(t, "", b.String())

	templ, err = repo.Get("T1")
	assert.Nil(t, err)
	err = templ.Execute(&b, nil)
	assert.Nil(t, err)

	assert.Equal(t, "T1", b.String())
}

type testData struct {
	Children []testData
	Name     string
	Recurse  bool
}

func TestRepoLoadsAllDependantTemplates(t *testing.T) {

	var b bytes.Buffer
	repo := NewRepository(nil)

	repo.AddFile("multiple", multipleDefinitions)
	repo.AddFile("dependant", dependantTemplate)

	templ, err := repo.Get("dependant")
	assert.Nil(t, err)

	err = templ.Execute(&b, nil)

	assert.Nil(t, err)

	assert.Equal(t, "T1D1", b.String())

}

func TestRepoRecursiveTemplates(t *testing.T) {

	var b bytes.Buffer
	repo := NewRepository(nil)

	repo.AddFile("c1", cirularDeps1)
	repo.AddFile("c2", cirularDeps2)

	templ, err := repo.Get("c1")
	assert.Nil(t, err)
	data := testData{
		Name: "Root",
		Children: []testData{
			{Recurse: false},
		},
	}
	expected := `Root: Children`
	err = templ.Execute(&b, data)

	assert.Nil(t, err)

	assert.Equal(t, expected, b.String())

	data = testData{
		Name: "Root",
		Children: []testData{
			{Name: "Child1", Recurse: true, Children: []testData{{Name: "Child2"}}},
		},
	}

	b.Reset()

	expected = `Root: Child1: Children`

	err = templ.Execute(&b, data)

	assert.Nil(t, err)

	assert.Equal(t, expected, b.String())

	data = testData{
		Name: "Root",
		Children: []testData{
			{Name: "Child1", Recurse: false, Children: []testData{{Name: "Child2"}}},
		},
	}

	b.Reset()

	expected = `Root: Children`

	err = templ.Execute(&b, data)

	assert.Nil(t, err)

	assert.Equal(t, expected, b.String())
}
