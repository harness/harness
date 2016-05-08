package yaml

// Build represents Docker image build instructions.
type Build struct {
	Context    string
	Dockerfile string
	Args       map[string]string
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (b *Build) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&b.Context)
	if err == nil {
		return nil
	}
	out := struct {
		Context    string
		Dockerfile string
		Args       map[string]string
	}{}
	err = unmarshal(&out)
	b.Context = out.Context
	b.Args = out.Args
	b.Dockerfile = out.Dockerfile
	return err
}
