// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// We need to compile the templates because this is no longer done in init
func TestMain(m *testing.M) {
	compileTemplates()
	retCode := m.Run()
	os.Exit(retCode)
}

var customHeader = `custom header`
var customMultiple = `{{define "bindprimitiveparam" }}custom primitive{{end}}`
var customNewTemplate = `new template`
var customExistingUsesNew = `{{define "bindprimitiveparam" }}{{ template "newtemplate" }}{{end}}`

func TestCustomTemplates(t *testing.T) {

	var buf bytes.Buffer
	headerTempl, err := templates.Get("bindprimitiveparam")

	assert.Nil(t, err)

	err = headerTempl.Execute(&buf, nil)

	assert.Nil(t, err)
	assert.Equal(t, "\n", buf.String())

	buf.Reset()
	err = templates.AddFile("bindprimitiveparam", customHeader)

	assert.Nil(t, err)
	headerTempl, err = templates.Get("bindprimitiveparam")

	assert.Nil(t, err)

	err = headerTempl.Execute(&buf, nil)

	assert.Nil(t, err)
	assert.Equal(t, "custom header", buf.String())

}

func TestCustomTemplatesMultiple(t *testing.T) {
	var buf bytes.Buffer

	err := templates.AddFile("differentFileName", customMultiple)

	assert.Nil(t, err)
	headerTempl, err := templates.Get("bindprimitiveparam")

	assert.Nil(t, err)

	err = headerTempl.Execute(&buf, nil)

	assert.Nil(t, err)
	assert.Equal(t, "custom primitive", buf.String())
}

func TestCustomNewTemplates(t *testing.T) {
	var buf bytes.Buffer

	err := templates.AddFile("newtemplate", customNewTemplate)
	err = templates.AddFile("existingUsesNew", customExistingUsesNew)

	assert.Nil(t, err)
	headerTempl, err := templates.Get("bindprimitiveparam")

	assert.Nil(t, err)

	err = headerTempl.Execute(&buf, nil)

	assert.Nil(t, err)
	assert.Equal(t, "new template", buf.String())
}
