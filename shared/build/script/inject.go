package script

import (
	"strings"
)

func Inject(script string, params map[string]string) string {
	if params == nil {
		return script
	}
	injected := script
	for k, v := range params {
		injected = strings.Replace(injected, "$$"+k, v, -1)
	}
	return injected
}
