package main

import (
	"fmt"
	"io/ioutil"

	"github.com/codegangsta/cli"
	"github.com/drone/drone/client"
)

// NewSetKeyCommand returns the CLI command for "set-key".
func NewSetKeyCommand() cli.Command {
	return cli.Command{
		Name:  "set-key",
		Usage: "sets the SSH private key used to clone",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {
			handle(c, setKeyCommandFunc)
		},
	}
}

// setKeyCommandFunc executes the "set-key" command.
func setKeyCommandFunc(c *cli.Context, client *client.Client) error {
	var host, owner, name, path string
	var args = c.Args()

	if len(args) != 0 {
		host, owner, name = parseRepo(args[0])
	}

	if len(args) == 2 {
		path = args[1]
	}

	pub, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Could not find private RSA key %s. %s", path, err)
	}

	path_pub := path + ".pub"
	priv, err := ioutil.ReadFile(path_pub)
	if err != nil {
		return fmt.Errorf("Could not find public RSA key %s. %s", path_pub, err)
	}

	return client.Repos.SetKey(host, owner, name, string(pub), string(priv))
}
