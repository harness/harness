package yaml

import "gopkg.in/yaml.v2"

// ParseBranch parses the branch section of the Yaml document.
func ParseBranch(in []byte) Constraint {
	out := struct {
		Constraint Constraint `yaml:"branches"`
	}{}

	yaml.Unmarshal(in, &out)
	return out.Constraint
}

// ParseBranchString parses the branch section of the Yaml document.
func ParseBranchString(in string) Constraint {
	return ParseBranch([]byte(in))
}
