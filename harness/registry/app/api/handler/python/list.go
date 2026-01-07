//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/request"
)

const HTMLTemplate = `
<!DOCTYPE html>
<html>
	<head>
		<meta name="pypi:repository-version" content="1.3">
		<title>Links for {{.Name}}</title>
	</head>
	<body>
		{{- /* PEP 503 – Simple Repository API: https://peps.python.org/pep-0503/ */ -}}
		<h1>Links for {{.Name}}</h1>
			{{range .Files}}
				<a href="{{.FileURL}}"{{if .RequiresPython}} data-requires-python="{{.RequiresPython}}"{{end}}>{{.Name}}</a><br>
			{{end}}
	</body>
</html>
`

func (h *handler) PackageMetadata(w http.ResponseWriter, r *http.Request) {
	contextInfo := request.ArtifactInfoFrom(r.Context())
	info, ok := contextInfo.(*pythontype.ArtifactInfo)
	if !ok {
		render.TranslatedUserError(r.Context(), w, fmt.Errorf("invalid request context"))
		return
	}

	packageData := h.controller.GetPackageMetadata(r.Context(), *info)
	if packageData.GetError() != nil {
		notFound := errors.IsErr(packageData.GetError(), errors.NotFoundCode)
		if notFound {
			render.NotFound(r.Context(), w)
			return
		}
		render.TranslatedUserError(r.Context(), w, packageData.GetError())
		return
	}

	// Parse and execute the template
	tmpl, err := template.New("simple").Parse(HTMLTemplate)
	if err != nil {
		render.TranslatedUserError(r.Context(), w, fmt.Errorf("template error: %w", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, packageData.PackageMetadata); err != nil {
		render.TranslatedUserError(r.Context(), w, fmt.Errorf("template rendering error: %w", err))
	}
}
