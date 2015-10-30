// +build ignore

// This program generates api documentation from a
// swaggerfile using an amber template.

package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/eknkc/amber"
	"github.com/go-swagger/go-swagger/spec"
)

var (
	templ  = flag.String("template", "index.amber", "")
	input  = flag.String("input", "swagger.json", "")
	output = flag.String("output", "", "")
)

func main() {
	flag.Parse()

	// parses the swagger spec file
	spec, err := spec.YAMLSpec(*input)
	if err != nil {
		log.Fatal(err)
	}
	swag := spec.Spec()

	// create output source for file. defaults to
	// stdout but may be file.
	var w io.WriteCloser = os.Stdout
	if *output != "" {
		w, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		defer w.Close()
	}

	// we wrap the swagger file in a map, otherwise it
	// won't work with our existing templates, which expect
	// a map as the root parameter.
	var data = map[string]interface{}{
		"Swagger": normalize(swag),
	}

	t := amber.MustCompileFile(*templ, amber.DefaultOptions)
	err = t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

// Swagger is a simplified representation of the swagger
// document with a subset of the fields used to generate
// our API documentation.
type Swagger struct {
	Tags []Tag
}

type Tag struct {
	Name string
	Ops  []Operation
}

type Operation struct {
	ID      string
	Method  string
	Path    string
	Desc    string
	Summary string

	Params  []Param
	Results []Result
}

type Param struct {
	Name     string
	Desc     string
	Type     string
	Example  interface{}
	InputTo  string
	IsObject bool
}

type Result struct {
	Status   int
	Desc     string
	Example  interface{}
	IsObject bool
	IsArray  bool
}

// normalize is a helper function that normalizes the swagger
// file to a simpler format that makes it easier to work with
// inside the template.
func normalize(swag *spec.Swagger) Swagger {
	swag_ := Swagger{}

	for _, tag := range swag.Tags {
		tag_ := Tag{Name: tag.Name}

		// group the paths based on their tag value.
		for route, path := range swag.Paths.Paths {

			var ops = []*spec.Operation{
				path.Get,
				path.Put,
				path.Post,
				path.Patch,
				path.Delete,
			}

			// flatten the operations into an array and convert
			// the underlying data so that it is a bit easier to
			// work with.
			for _, op := range ops {

				// the operation must have a tag to
				// be rendered in our custom template.
				if op == nil || !hasTag(tag.Name, op.Tags) {
					continue
				}

				item := Operation{}
				item.Path = route
				item.Method = getMethod(op, path)
				item.Desc = op.Description
				item.Summary = op.Summary
				item.ID = fmt.Sprintf("%x", md5.Sum([]byte(item.Path+item.Method)))

				// convert the operation input parameters into
				// our internal format so that it is easier to
				// work with in the template.
				for _, param := range op.Parameters {
					param_ := Param{}
					param_.Name = param.Name
					param_.Desc = param.Description
					param_.Type = param.Type
					param_.IsObject = param.Schema != nil
					param_.InputTo = param.In

					if param_.IsObject {
						param_.Type = param.Schema.Ref.GetPointer().String()[13:]
						param_.Example = param.Schema.Example
					}
					item.Params = append(item.Params, param_)
				}

				// convert the operation response types into
				// our internal format so that it is easier to
				// work with in the template.
				for code, resp := range op.Responses.StatusCodeResponses {
					result := Result{}
					result.Desc = resp.Description
					result.Status = code
					result.IsObject = resp.Schema != nil
					if result.IsObject {
						result.IsArray = resp.Schema.Items != nil

						name := resp.Schema.Ref.GetPointer().String()
						if len(name) != 0 {
							def, _ := swag.Definitions[name[13:]]
							result.Example = def.Example
						}
					}
					if result.IsArray {
						name := resp.Schema.Items.Schema.Ref.GetPointer().String()
						def, _ := swag.Definitions[name[13:]]
						result.Example = def.Example
					}
					item.Results = append(item.Results, result)
				}
				sort.Sort(ByCode(item.Results))
				tag_.Ops = append(tag_.Ops, item)
			}
		}

		sort.Sort(ByPath(tag_.Ops))
		swag_.Tags = append(swag_.Tags, tag_)
	}

	return swag_
}

// hasTag is a helper function that returns true if
// an operation has the specified tag label.
func hasTag(want string, in []string) bool {
	for _, got := range in {
		if got == want {
			return true
		}
	}
	return false
}

// getMethod is a helper function that returns the http
// method for the specified operation in a path.
func getMethod(op *spec.Operation, path spec.PathItem) string {
	switch {
	case op == path.Get:
		return "GET"
	case op == path.Put:
		return "PUT"
	case op == path.Patch:
		return "PATCH"
	case op == path.Post:
		return "POST"
	case op == path.Delete:
		return "DELETE"
	}
	return ""
}

// ByCode helps sort a list of results by status code
type ByCode []Result

func (a ByCode) Len() int           { return len(a) }
func (a ByCode) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCode) Less(i, j int) bool { return a[i].Status < a[j].Status }

// ByPath helps sort a list of endpoints by path
type ByPath []Operation

func (a ByPath) Len() int           { return len(a) }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }
