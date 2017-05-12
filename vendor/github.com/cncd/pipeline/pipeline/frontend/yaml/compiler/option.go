package compiler

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cncd/pipeline/pipeline/frontend"
)

// Option configures a compiler option.
type Option func(*Compiler)

// WithOption configures the compiler with the given option if
// boolean b evaluates to true.
func WithOption(option Option, b bool) Option {
	switch {
	case b:
		return option
	default:
		return func(compiler *Compiler) {}
	}
}

// WithVolumes configutes the compiler with default volumes that
// are mounted to each container in the pipeline.
func WithVolumes(volumes ...string) Option {
	return func(compiler *Compiler) {
		compiler.volumes = volumes
	}
}

// WithRegistry configures the compiler with registry credentials
// that should be used to download images.
func WithRegistry(registries ...Registry) Option {
	return func(compiler *Compiler) {
		compiler.registries = registries
	}
}

// WithSecret configures the compiler with external secrets
// to be injected into the container at runtime.
func WithSecret(secrets ...Secret) Option {
	return func(compiler *Compiler) {
		for _, secret := range secrets {
			compiler.secrets[strings.ToLower(secret.Name)] = secret
		}
	}
}

// WithMetadata configutes the compiler with the repostiory, build
// and system metadata. The metadata is used to remove steps from
// the compiled pipeline configuration that should be skipped. The
// metadata is also added to each container as environment variables.
func WithMetadata(metadata frontend.Metadata) Option {
	return func(compiler *Compiler) {
		compiler.metadata = metadata

		for k, v := range metadata.Environ() {
			compiler.env[k] = v
		}
		// TODO this is present for backward compatibility and should
		// be removed in a future version.
		for k, v := range metadata.EnvironDrone() {
			compiler.env[k] = v
		}
	}
}

// WithNetrc configures the compiler with netrc authentication
// credentials added by default to every container in the pipeline.
func WithNetrc(username, password, machine string) Option {
	return WithEnviron(
		map[string]string{
			"CI_NETRC_USERNAME": username,
			"CI_NETRC_PASSWORD": password,
			"CI_NETRC_MACHINE":  machine,

			// TODO: This is present for backward compatibility and should
			// be removed in a future version.
			"DRONE_NETRC_USERNAME": username,
			"DRONE_NETRC_PASSWORD": password,
			"DRONE_NETRC_MACHINE":  machine,
		},
	)
}

// WithWorkspace configures the compiler with the workspace base
// and path. The workspace base is a volume created at runtime and
// mounted into all containers in the pipeline. The base and path
// are joined to provide the working directory for all build and
// plugin steps in the pipeline.
func WithWorkspace(base, path string) Option {
	return func(compiler *Compiler) {
		compiler.base = base
		compiler.path = path
	}
}

// WithWorkspaceFromURL configures the compiler with the workspace
// base and path based on the repository url.
func WithWorkspaceFromURL(base, link string) Option {
	path := "src"
	parsed, err := url.Parse(link)
	if err == nil {
		path = filepath.Join(path, parsed.Host, parsed.Path)
	}
	return WithWorkspace(base, path)
}

// WithEscalated configures the compiler to automatically execute
// images as privileged containers if the match the given list.
func WithEscalated(images ...string) Option {
	return func(compiler *Compiler) {
		compiler.escalated = images
	}
}

// WithPrefix configures the compiler with the prefix. The prefix is
// used to prefix container, volume and network names to avoid
// collision at runtime.
func WithPrefix(prefix string) Option {
	return func(compiler *Compiler) {
		compiler.prefix = prefix
	}
}

// WithLocal configures the compiler with the local flag. The local
// flag indicates the pipeline execution is running in a local development
// environment with a mounted local working directory.
func WithLocal(local bool) Option {
	return func(compiler *Compiler) {
		compiler.local = local
	}
}

// WithEnviron configures the compiler with environment variables
// added by default to every container in the pipeline.
func WithEnviron(env map[string]string) Option {
	return func(compiler *Compiler) {
		for k, v := range env {
			compiler.env[k] = v
		}
	}
}

// WithProxy configures the compiler with HTTP_PROXY, HTTPS_PROXY,
// and NO_PROXY environment variables added by default to every
// container in the pipeline.
func WithProxy() Option {
	return WithEnviron(
		map[string]string{
			"no_proxy":    noProxy,
			"NO_PROXY":    noProxy,
			"http_proxy":  httpProxy,
			"HTTP_PROXY":  httpProxy,
			"HTTPS_PROXY": httpsProxy,
			"https_proxy": httpsProxy,
		},
	)
}

// WithNetworks configures the compiler with additionnal networks
// to be connected to build containers
func WithNetworks(networks ...string) Option {
	return func(compiler *Compiler) {
		compiler.networks = networks
	}
}

// TODO(bradrydzewski) consider an alternate approach to
// WithProxy where the proxy strings are passed directly
// to the function as named parameters.

// func WithProxy2(http, https, none string) Option {
// 	return WithEnviron(
// 		map[string]string{
// 			"no_proxy":    none,
// 			"NO_PROXY":    none,
// 			"http_proxy":  http,
// 			"HTTP_PROXY":  http,
// 			"HTTPS_PROXY": https,
// 			"https_proxy": https,
// 		},
// 	)
// }

var (
	noProxy    = getenv("no_proxy")
	httpProxy  = getenv("https_proxy")
	httpsProxy = getenv("https_proxy")
)

// getenv returns the named environment variable.
func getenv(name string) (value string) {
	name = strings.ToUpper(name)
	if value := os.Getenv(name); value != "" {
		return value
	}
	name = strings.ToLower(name)
	if value := os.Getenv(name); value != "" {
		return value
	}
	return
}
