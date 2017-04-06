package main

import "github.com/urfave/cli"

var registryCmd = cli.Command{
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
