package agent

import (
	"strings"

	"github.com/drone/drone/model"
)

// SecretReplacer hides secrets from being exposed by the build output.
type SecretReplacer interface {
	// Replace conceals instances of secrets found in s.
	Replace(s string) string
}

// NewSecretReplacer creates a SecretReplacer based on whether any value in
// secrets requests it be hidden.
func NewSecretReplacer(secrets []*model.Secret) SecretReplacer {
	var r []string
	for _, s := range secrets {
		if s.Conceal {
			r = append(r, s.Value, "*****")
		}
	}

	if len(r) == 0 {
		return &noopReplacer{}
	}

	return &secretReplacer{
		replacer: strings.NewReplacer(r...),
	}
}

type noopReplacer struct{}

func (*noopReplacer) Replace(s string) string {
	return s
}

type secretReplacer struct {
	replacer *strings.Replacer
}

func (r *secretReplacer) Replace(s string) string {
	return r.replacer.Replace(s)
}
