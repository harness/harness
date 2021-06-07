package jsonnet

import (
	"bytes"
	"fmt"

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
	var jsonnetFileName string
	if template != nil {
		jsonnetFile = template.Data
		jsonnetFileName = template.Name
	} else {
		jsonnetFile = req.Config.Data
		jsonnetFileName = req.Repo.Config
	}
	// map external inputs
	if len(templateData) != 0 {
		for k, v := range templateData {
			key := fmt.Sprintf("input." + k)
			val := fmt.Sprint(v)
			vm.ExtVar(key, val)
		}
	}
	// convert the jsonnet file to yaml
	buf := new(bytes.Buffer)
	docs, err := vm.EvaluateSnippetStream(jsonnetFileName, jsonnetFile)
	if err != nil {
		doc, err2 := vm.EvaluateSnippet(jsonnetFileName, jsonnetFile)
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
