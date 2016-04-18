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
	"encoding/json"
	"regexp"
	"text/template"
)

//go:generate go-bindata -pkg=generator -ignore=.*\.sw? ./templates/...

// fwiw, don't get attached to this, still requires a better abstraction

var (
	modelTemplate *template.Template
	// modelValidatorTemplate *template.Template
	operationTemplate      *template.Template
	parameterTemplate      *template.Template
	responsesTemplate      *template.Template
	builderTemplate        *template.Template
	serverTemplate         *template.Template
	mainTemplate           *template.Template
	mainDocTemplate        *template.Template
	embeddedSpecTemplate   *template.Template
	configureAPITemplate   *template.Template
	clientTemplate         *template.Template
	clientParamTemplate    *template.Template
	clientResponseTemplate *template.Template
	clientFacadeTemplate   *template.Template
)

var assets = map[string][]byte{
	"validation/primitive.gotmpl":           MustAsset("templates/validation/primitive.gotmpl"),
	"validation/customformat.gotmpl":        MustAsset("templates/validation/customformat.gotmpl"),
	"docstring.gotmpl":                      MustAsset("templates/docstring.gotmpl"),
	"validation/structfield.gotmpl":         MustAsset("templates/validation/structfield.gotmpl"),
	"modelvalidator.gotmpl":                 MustAsset("templates/modelvalidator.gotmpl"),
	"structfield.gotmpl":                    MustAsset("templates/structfield.gotmpl"),
	"tupleserializer.gotmpl":                MustAsset("templates/tupleserializer.gotmpl"),
	"additionalpropertiesserializer.gotmpl": MustAsset("templates/additionalpropertiesserializer.gotmpl"),
	"schematype.gotmpl":                     MustAsset("templates/schematype.gotmpl"),
	"schemabody.gotmpl":                     MustAsset("templates/schemabody.gotmpl"),
	"schema.gotmpl":                         MustAsset("templates/schema.gotmpl"),
	"schemavalidator.gotmpl":                MustAsset("templates/schemavalidator.gotmpl"),
	"model.gotmpl":                          MustAsset("templates/model.gotmpl"),
	"header.gotmpl":                         MustAsset("templates/header.gotmpl"),
	"swagger_json_embed.gotmpl":             MustAsset("templates/swagger_json_embed.gotmpl"),

	"server/parameter.gotmpl":    MustAsset("templates/server/parameter.gotmpl"),
	"server/responses.gotmpl":    MustAsset("templates/server/responses.gotmpl"),
	"server/operation.gotmpl":    MustAsset("templates/server/operation.gotmpl"),
	"server/builder.gotmpl":      MustAsset("templates/server/builder.gotmpl"),
	"server/server.gotmpl":       MustAsset("templates/server/server.gotmpl"),
	"server/configureapi.gotmpl": MustAsset("templates/server/configureapi.gotmpl"),
	"server/main.gotmpl":         MustAsset("templates/server/main.gotmpl"),
	"server/doc.gotmpl":          MustAsset("templates/server/doc.gotmpl"),

	"client/parameter.gotmpl": MustAsset("templates/client/parameter.gotmpl"),
	"client/response.gotmpl":  MustAsset("templates/client/response.gotmpl"),
	"client/client.gotmpl":    MustAsset("templates/client/client.gotmpl"),
	"client/facade.gotmpl":    MustAsset("templates/client/facade.gotmpl"),
}

var (
	notNumberExp = regexp.MustCompile("[^0-9]")
)

var templates = NewRepository(FuncMap)

func init() {

	templates.LoadDefaults()

}

func compileTemplates() {

	modelTemplate = template.Must(templates.Get("model"))

	// server templates
	parameterTemplate = template.Must(templates.Get("serverParameter"))

	responsesTemplate = template.Must(templates.Get("serverResponses"))

	operationTemplate = template.Must(templates.Get("serverOperation"))
	builderTemplate = template.Must(templates.Get("serverBuilder"))

	serverTemplate = template.Must(templates.Get("serverServer"))

	configureAPITemplate = template.Must(templates.Get("serverConfigureapi"))
	mainTemplate = template.Must(templates.Get("serverMain"))
	mainDocTemplate = template.Must(templates.Get("serverDoc"))

	embeddedSpecTemplate = template.Must(templates.Get("swaggerJsonEmbed"))

	// Client templates
	clientParamTemplate = template.Must(templates.Get("clientParameter"))

	clientResponseTemplate = template.Must(templates.Get("clientResponse"))

	clientTemplate = template.Must(templates.Get("clientClient"))

	clientFacadeTemplate = template.Must(templates.Get("clientFacade"))

}

func asJSON(data interface{}) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func asPrettyJSON(data interface{}) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
