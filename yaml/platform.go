package yaml

import "gopkg.in/yaml.v2"

// ParsePlatform parses the platform section of the Yaml document.
func ParsePlatform(in []byte) string {
	out := struct {
		Platform string `yaml:"platform"`
	}{}

	yaml.Unmarshal(in, &out)
	return out.Platform
}

// ParsePlatformString parses the platform section of the Yaml document.
func ParsePlatformString(in string) string {
	return ParsePlatform([]byte(in))
}

// ParsePlatformDefault parses the platform section of the Yaml document.
func ParsePlatformDefault(in []byte, platform string) string {
	if p := ParsePlatform([]byte(in)); p != "" {
		return p
	}
	return platform
}
