package builtin

import (
	"os"
	"strings"

	"github.com/drone/drone/engine/compiler/parse"
)

var (
	httpProxy  = os.Getenv("HTTP_PROXY")
	httpsProxy = os.Getenv("HTTPS_PROXY")
	noProxy    = os.Getenv("NO_PROXY")
)

type envOp struct {
	visitor
	envs map[string]string
}

// NewEnvOp returns a transformer that sets default environment variables
// for each container, service and plugin.
func NewEnvOp(envs map[string]string) Visitor {
	return &envOp{
		envs: envs,
	}
}

func (v *envOp) VisitContainer(node *parse.ContainerNode) error {
	if node.Container.Environment == nil {
		node.Container.Environment = map[string]string{}
	}
	v.defaultEnv(node)
	v.defaultEnvProxy(node)
	return nil
}

func (v *envOp) defaultEnv(node *parse.ContainerNode) {
	for k, v := range v.envs {
		node.Container.Environment[k] = v
	}
}

func (v *envOp) defaultEnvProxy(node *parse.ContainerNode) {
	if httpProxy != "" {
		node.Container.Environment["HTTP_PROXY"] = httpProxy
		node.Container.Environment["http_proxy"] = strings.ToUpper(httpProxy)
	}
	if httpsProxy != "" {
		node.Container.Environment["HTTPS_PROXY"] = httpsProxy
		node.Container.Environment["https_proxy"] = strings.ToUpper(httpsProxy)
	}
	if noProxy != "" {
		node.Container.Environment["NO_PROXY"] = noProxy
		node.Container.Environment["no_proxy"] = strings.ToUpper(noProxy)
	}
}
