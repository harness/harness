package compiler

import "github.com/drone/drone/engine/compiler/parse"

// Transform is used to transform nodes from the parsed Yaml file during the
// compilation process. A Transform may be used to add, disable or alter nodes.
type Transform interface {
	VisitRoot(*parse.RootNode) error
	VisitVolume(*parse.VolumeNode) error
	VisitNetwork(*parse.NetworkNode) error
	VisitBuild(*parse.BuildNode) error
	VisitContainer(*parse.ContainerNode) error
}
