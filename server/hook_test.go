package server

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestMultilineEnvsubst(t *testing.T) {
	b := builder{
		Repo: &model.Repo{},
		Curr: &model.Build{
			Message: `aaa
bbb`,
		},
		Last: &model.Build{},
		Netrc: &model.Netrc{},
		Secs: []*model.Secret{},
		Regs: []*model.Registry{},
		Link: "",
		Yaml: `pipeline:
  xxx:
    image: scratch
    yyy: ${DRONE_COMMIT_MESSAGE}
`,
	}

	if _, err := b.Build(); err != nil {
		t.Fatal(err)
	}
}
