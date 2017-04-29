package registry

import "github.com/urfave/cli"

// Command exports the registry command set.
var Command = cli.Command{
	Name:  "registry",
	Usage: "manage registries",
	Subcommands: []cli.Command{
		registryCreateCmd,
		registryDeleteCmd,
		registryUpdateCmd,
		registryInfoCmd,
		registryListCmd,
	},
}
