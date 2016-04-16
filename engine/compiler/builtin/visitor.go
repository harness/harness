package builtin

import "github.com/drone/drone/engine/compiler/parse"

// Visitor interface for walking the Yaml file.
type Visitor interface {
	VisitRoot(*parse.RootNode) error
	VisitVolume(*parse.VolumeNode) error
	VisitNetwork(*parse.NetworkNode) error
	VisitBuild(*parse.BuildNode) error
	VisitContainer(*parse.ContainerNode) error
}

// visitor provides an easy default implementation of a Visitor interface with
// stubbed methods. This can be embedded in transforms to meet the basic
// requirements.
type visitor struct{}

func (visitor) VisitRoot(*parse.RootNode) error           { return nil }
func (visitor) VisitVolume(*parse.VolumeNode) error       { return nil }
func (visitor) VisitNetwork(*parse.NetworkNode) error     { return nil }
func (visitor) VisitBuild(*parse.BuildNode) error         { return nil }
func (visitor) VisitContainer(*parse.ContainerNode) error { return nil }
