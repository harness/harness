package yaml

import (
	"gopkg.in/yaml.v2"

	"github.com/drone/drone/yaml/types"
)

// ParseLabel parses the labels section of the Yaml document.
func ParseLabel(in []byte) map[string]string {
	out := struct {
		Labels types.MapEqualSlice `yaml:"labels"`
	}{}

	yaml.Unmarshal(in, &out)
	labels := out.Labels.Map()
	if labels == nil {
		labels = make(map[string]string)
	}
	return labels
}

// ParseLabelString parses the labels section of the Yaml document.
func ParseLabelString(in string) map[string]string {
	return ParseLabel([]byte(in))
}
