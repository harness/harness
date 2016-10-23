package yaml

import "gopkg.in/yaml.v2"

// ParseBranch parses the branch section of the Yaml document.
func ParseLabel(in []byte) string {
	out := struct {
		Label string `yaml:"label"`
	}{}

	yaml.Unmarshal(in, &out)
	return out.Label
}

// ParseBranchString parses the branch section of the Yaml document.
func ParseLabelString(in string) string {
	return ParseLabel([]byte(in))
}
