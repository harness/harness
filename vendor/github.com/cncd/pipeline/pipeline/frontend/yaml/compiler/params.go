package compiler

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	json "github.com/ghodss/yaml"
	"gopkg.in/yaml.v2"
)

// paramsToEnv uses reflection to convert a map[string]interface to a list
// of environment variables.
func paramsToEnv(from map[string]interface{}, to map[string]string) error {
	for k, v := range from {
		if v == nil {
			continue
		}

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

		case reflect.Map:
			yml, _ := yaml.Marshal(vv.Interface())
			out, _ := json.YAMLToJSON(yml)
			to[k] = string(out)

		case reflect.Slice:
			out, err := yaml.Marshal(vv.Interface())
			if err != nil {
				return err
			}

			in := []string{}
			err = yaml.Unmarshal(out, &in)
			if err == nil {
				to[k] = strings.Join(in, ",")
			} else {
				out, err = json.YAMLToJSON(out)
				if err != nil {
					return err
				}
				to[k] = string(out)
			}
		}
	}

	return nil
}
