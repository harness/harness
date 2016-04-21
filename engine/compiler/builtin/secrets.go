package builtin

import (
	"github.com/drone/drone/engine/compiler/parse"
	"github.com/drone/drone/model"
)

type secretOp struct {
	visitor
	event   string
	secrets []*model.Secret
}

// NewSecretOp returns a transformer that configures plugin secrets.
func NewSecretOp(event string, secrets []*model.Secret) Visitor {
	return &secretOp{
		event:   event,
		secrets: secrets,
	}
}

func (v *secretOp) VisitContainer(node *parse.ContainerNode) error {
	for _, secret := range v.secrets {
		if !secret.Match(node.Container.Image, v.event) {
			continue
		}
		if node.Container.Environment == nil {
			node.Container.Environment = map[string]string{}
		}
		node.Container.Environment[secret.Name] = secret.Value
	}
	return nil
}
