package agent

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

const testString = "This is SECRET: secret_value"

func TestSecret(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("SecretReplacer", func() {
		g.It("Should conceal secret", func() {
			secrets := []*model.Secret{
				{
					Name:    "SECRET",
					Value:   "secret_value",
					Conceal: true,
				},
			}
			r := NewSecretReplacer(secrets)
			g.Assert(r.Replace(testString)).Equal("This is SECRET: *****")
		})

		g.It("Should not conceal secret", func() {
			secrets := []*model.Secret{
				{
					Name:    "SECRET",
					Value:   "secret_value",
					Conceal: false,
				},
			}
			r := NewSecretReplacer(secrets)
			g.Assert(r.Replace(testString)).Equal(testString)
		})
	})
}
