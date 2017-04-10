package yaml

type (
	// Secrets defines a collection of secrets.
	Secrets struct {
		Secrets []*Secret
	}

	// Secret defines a container secret.
	Secret struct {
		Source string `yaml:"source"`
		Target string `yaml:"target"`
	}
)

// UnmarshalYAML implements the Unmarshaller interface.
func (s *Secrets) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strslice []string
	err := unmarshal(&strslice)
	if err == nil {
		for _, str := range strslice {
			s.Secrets = append(s.Secrets, &Secret{
				Source: str,
				Target: str,
			})
		}
		return nil
	}
	return unmarshal(&s.Secrets)
}
