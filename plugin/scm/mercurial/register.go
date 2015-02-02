package mercurial

import (
	"github.com/drone/drone/plugin/scm"
)

func Register() {
	scm.Register(
		New(),
	)
}
