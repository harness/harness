package local

import (
	"github.com/drone/drone/plugin/scm"
)

func Register() {
	scm.Register(
		New(),
	)
}
