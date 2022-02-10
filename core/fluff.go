package core

import "github.com/drone/runner-go/manifest"

type Pipeline struct {
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Type    string `json:"type,omitempty"`
	Name    string `json:"name,omitempty"`

	Concurrency manifest.Concurrency `json:"concurrency,omitempty"`
	DependsOn   []string             `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Node        map[string]string    `json:"node,omitempty" yaml:"node"`
	Platform    manifest.Platform    `json:"platform,omitempty"`
	PullSecrets []string             `json:"pull_secrets,omitempty" yaml:"pull_secrets,omitempty"`
	Trigger     manifest.Conditions  `json:"trigger,omitempty"`
}

func (p *Pipeline) GetKind() string    { return p.Kind }
func (p *Pipeline) GetVersion() string { return p.Version }
func (p *Pipeline) GetName() string    { return p.Name }
func (p *Pipeline) GetType() string    { return p.Type }
