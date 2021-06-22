package jsonnet

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/drone/drone/core"

	"github.com/google/go-jsonnet"
)

const repo = "repo."
const build = "build."
const param = "param."

func Parse(req *core.ConvertArgs, template *core.Template, templateData map[string]interface{}) (string, error) {
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500
	vm.StringOutput = false
	vm.ErrorFormatter.SetMaxStackTraceSize(20)

	//map build/repo parameters
	if req.Build != nil {
		mapBuild(req.Build, vm)
	}
	if req.Repo != nil {
		mapRepo(req.Repo, vm)
	}

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

func mapBuild(v *core.Build, vm *jsonnet.VM) {
	vm.ExtVar(build+"event", v.Event)
	vm.ExtVar(build+"action", v.Action)
	vm.ExtVar(build+"environment", v.Deploy)
	vm.ExtVar(build+"link", v.Link)
	vm.ExtVar(build+"branch", v.Target)
	vm.ExtVar(build+"source", v.Source)
	vm.ExtVar(build+"before", v.Before)
	vm.ExtVar(build+"after", v.After)
	vm.ExtVar(build+"target", v.Target)
	vm.ExtVar(build+"ref", v.Ref)
	vm.ExtVar(build+"commit", v.After)
	vm.ExtVar(build+"ref", v.Ref)
	vm.ExtVar(build+"title", v.Title)
	vm.ExtVar(build+"message", v.Message)
	vm.ExtVar(build+"source_repo", v.Fork)
	vm.ExtVar(build+"author_login", v.Author)
	vm.ExtVar(build+"author_name", v.AuthorName)
	vm.ExtVar(build+"author_email", v.AuthorEmail)
	vm.ExtVar(build+"author_avatar", v.AuthorAvatar)
	vm.ExtVar(build+"sender", v.Sender)
	fromMap(v.Params, vm)
}

func mapRepo(v *core.Repository, vm *jsonnet.VM) {
	vm.ExtVar(repo+"uid", v.UID)
	vm.ExtVar(repo+"name", v.Name)
	vm.ExtVar(repo+"namespace", v.Namespace)
	vm.ExtVar(repo+"slug", v.Slug)
	vm.ExtVar(repo+"git_http_url", v.HTTPURL)
	vm.ExtVar(repo+"git_ssh_url", v.SSHURL)
	vm.ExtVar(repo+"link", v.Link)
	vm.ExtVar(repo+"branch", v.Branch)
	vm.ExtVar(repo+"config", v.Config)
	vm.ExtVar(repo+"private", strconv.FormatBool(v.Private))
	vm.ExtVar(repo+"visibility", v.Visibility)
	vm.ExtVar(repo+"active", strconv.FormatBool(v.Active))
	vm.ExtVar(repo+"trusted", strconv.FormatBool(v.Trusted))
	vm.ExtVar(repo+"protected", strconv.FormatBool(v.Protected))
	vm.ExtVar(repo+"ignore_forks", strconv.FormatBool(v.IgnoreForks))
	vm.ExtVar(repo+"ignore_pull_requests", strconv.FormatBool(v.IgnorePulls))
}

func fromMap(m map[string]string, vm *jsonnet.VM) {
	for k, v := range m {
		vm.ExtVar(build+param+k, v)
	}
}
