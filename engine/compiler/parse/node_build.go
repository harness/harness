package parse

// BuildNode represents Docker image build instructions.
type BuildNode struct {
	NodeType

	Context    string
	Dockerfile string
	Args       map[string]string

	root *RootNode
}

// Root returns the root node.
func (n *BuildNode) Root() *RootNode { return n.root }

//
// intermediate types for yaml decoding.
//

type build struct {
	Context    string
	Dockerfile string
	Args       map[string]string
}

func (b *build) UnmarshalYAML(unmarshal func(interface{}) error) error {
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
