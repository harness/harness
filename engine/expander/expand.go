package expander

import "sort"

// Expand expands variables into the Yaml configuration using a
// ${key} template parameter with limited support for bash string functions.
func Expand(config []byte, envs map[string]string) []byte {
	return []byte(
		ExpandString(string(config), envs),
	)
}

// ExpandString injects the variables into the Yaml configuration string using
// a ${key} template parameter with limited support for bash string functions.
func ExpandString(config string, envs map[string]string) string {
	if envs == nil || len(envs) == 0 {
		return config
	}
	keys := []string{}
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	expanded := config
	for _, k := range keys {
		v := envs[k]

		for _, substitute := range substitutors {
			expanded = substitute(expanded, k, v)
		}
	}
	return expanded
}
