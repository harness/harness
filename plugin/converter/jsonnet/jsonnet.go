package jsonnet

import (
	"bytes"

	"github.com/drone/drone/core"

	"github.com/google/go-jsonnet"
)

func Parse(req *core.ConvertArgs, template *core.Template, templateData map[string]interface{}) (string, error) {
	// create the jsonnet vm
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500
	vm.StringOutput = false
	vm.ErrorFormatter.SetMaxStackTraceSize(20)

	var jsonnetFile string
	var jsonentFileName string
	if template != nil {
		jsonnetFile = template.Data
		jsonentFileName = template.Name
	} else {
		jsonnetFile = req.Config.Data
		jsonentFileName = req.Repo.Config
	}
	// map external inputs
	if len(templateData) != 0 {
		for k, v := range templateData {
			if s, ok := v.(string); ok {
				key := "input." + k
				vm.ExtVar(key, s)
			}

		}
	}
	// convert the jsonnet file to yaml
	buf := new(bytes.Buffer)
	docs, err := vm.EvaluateSnippetStream(jsonentFileName, jsonnetFile)
	if err != nil {
		doc, err2 := vm.EvaluateSnippet(jsonentFileName, jsonnetFile)
		if err2 != nil {
			return "", err
		}
		docs = append(docs, doc)
	}

	// the jsonnet vm returns a stream of yaml documents
	// that need to be combined into a single yaml file.
	for _, doc := range docs {
		buf.WriteString("---")
		buf.WriteString("\n")
		buf.WriteString(doc)
	}

	return buf.String(), nil
}
