package builtin

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/drone/drone/engine/compiler/parse"

	json "github.com/ghodss/yaml"
	"gopkg.in/yaml.v2"
)

type argsOps struct {
	visitor
}

// NewArgsOp returns a transformer that provides the plugin node
// with the custom arguments from the Yaml file.
func NewArgsOp() Visitor {
	return &argsOps{}
}

func (v *argsOps) VisitContainer(node *parse.ContainerNode) error {
	switch node.NodeType {
	case parse.NodePlugin, parse.NodeCache, parse.NodeClone:
		break // no-op
	default:
		return nil
	}
	if node.Container.Environment == nil {
		node.Container.Environment = map[string]string{}
	}
	return argsToEnv(node.Vargs, node.Container.Environment)
}

// argsToEnv uses reflection to convert a map[string]interface to a list
// of environment variables.
func argsToEnv(from map[string]interface{}, to map[string]string) error {

	for k, v := range from {
		t := reflect.TypeOf(v)
		vv := reflect.ValueOf(v)

		k = "PLUGIN_" + strings.ToUpper(k)

		switch t.Kind() {
		case reflect.Bool:
			to[k] = strconv.FormatBool(vv.Bool())

		case reflect.String:
			to[k] = vv.String()

		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
			to[k] = fmt.Sprintf("%v", vv.Int())

		case reflect.Float32, reflect.Float64:
			to[k] = fmt.Sprintf("%v", vv.Float())

		// case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
		// 	to[k] = strconv.FormatInt(vv.Int(), 16)

		// case reflect.Float32, reflect.Float64:
		// 	to[k] = strconv.FormatFloat(vv.Float(), 'E', -1, 64)

		case reflect.Map:
			yml, _ := yaml.Marshal(vv.Interface())
			out, _ := json.YAMLToJSON(yml)
			to[k] = string(out)

		case reflect.Slice:
			out, _ := yaml.Marshal(vv.Interface())

			in := []string{}
			err := yaml.Unmarshal(out, &in)
			if err == nil {
				to[k] = strings.Join(in, ",")
			} else {
				out, err = json.YAMLToJSON(out)
				if err != nil {
					println(err.Error())
				}
				to[k] = string(out)
			}
		}
	}

	return nil
}
