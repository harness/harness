package coding

import (
	"fmt"
)

func projectFullName(owner, name string) string {
	return fmt.Sprintf("%s/%s", owner, name)
}
