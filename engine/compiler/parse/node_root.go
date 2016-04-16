package parse

// RootNode is the root node in the parsed Yaml file.
type RootNode struct {
	NodeType

	Platform string
	Base     string
	Path     string
	Image    string

	Pod      Node
	Build    Node
	Cache    Node
	Clone    Node
	Script   []Node
	Volumes  []Node
	Networks []Node
	Services []Node
}

// NewRootNode returns a new root node.
func NewRootNode() *RootNode {
	return &RootNode{
		NodeType: NodeRoot,
	}
}

// Root returns the root node.
func (n *RootNode) Root() *RootNode { return n }

// Returns a new Volume Node.
func (n *RootNode) NewVolumeNode(name string) *VolumeNode {
	return &VolumeNode{
		NodeType: NodeVolume,
		Name:     name,
		root:     n,
	}
}

// Returns a new Network Node.
func (n *RootNode) NewNetworkNode(name string) *NetworkNode {
	return &NetworkNode{
		NodeType: NodeNetwork,
		Name:     name,
		root:     n,
	}
}

// Returns a new Network Node.
func (n *RootNode) NewBuildNode(context string) *BuildNode {
	return &BuildNode{
		NodeType: NodeBuild,
		Context:  context,
		root:     n,
	}
}

// Returns a new Container Plugin Node.
func (n *RootNode) NewPluginNode() *ContainerNode {
	return &ContainerNode{
		NodeType: NodePlugin,
		root:     n,
	}
}

// Returns a new Container Shell Node.
func (n *RootNode) NewShellNode() *ContainerNode {
	return &ContainerNode{
		NodeType: NodeShell,
		root:     n,
	}
}

// Returns a new Container Service Node.
func (n *RootNode) NewServiceNode() *ContainerNode {
	return &ContainerNode{
		NodeType: NodeService,
		root:     n,
	}
}

// Returns a new Container Clone Node.
func (n *RootNode) NewCloneNode() *ContainerNode {
	return &ContainerNode{
		NodeType: NodeClone,
		root:     n,
	}
}

// Returns a new Container Cache Node.
func (n *RootNode) NewCacheNode() *ContainerNode {
	return &ContainerNode{
		NodeType: NodeCache,
		root:     n,
	}
}

// Returns a new Container Node.
func (n *RootNode) NewContainerNode() *ContainerNode {
	return &ContainerNode{
		NodeType: NodeContainer,
		root:     n,
	}
}

// Walk is a function that walk through all child nodes of the RootNode
// and invokes the Walk callback function for each Node.
func (n *RootNode) Walk(fn WalkFunc) (err error) {
	var nodes []Node
	nodes = append(nodes, n)
	nodes = append(nodes, n.Build)
	nodes = append(nodes, n.Cache)
	nodes = append(nodes, n.Clone)
	nodes = append(nodes, n.Script...)
	nodes = append(nodes, n.Volumes...)
	nodes = append(nodes, n.Networks...)
	nodes = append(nodes, n.Services...)
	for _, node := range nodes {
		err = fn(node)
		if err != nil {
			return
		}
	}
	return
}

type WalkFunc func(Node) error

//
// intermediate types for yaml decoding.
//

type root struct {
	Workspace struct {
		Path string
		Base string
	}
	Image    string
	Platform string
	Volumes  volumeList
	Networks networkList
	Services containerList
	Script   containerList
	Cache    container
	Clone    container
	Build    build
}
