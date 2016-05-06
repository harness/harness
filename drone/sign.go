package main

import (
	"io/ioutil"
	"log"

	"github.com/codegangsta/cli"
)

var signCmd = cli.Command{
	Name:  "sign",
	Usage: "creates a secure yaml file",
	Action: func(c *cli.Context) {
		if err := sign(c); err != nil {
			log.Fatalln(err)
		}
	},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "in",
			Usage: "input file",
			Value: ".drone.yml",
		},
		cli.StringFlag{
			Name:  "out",
			Usage: "output file signature",
			Value: ".drone.yml.sig",
		},
	},
}

func sign(c *cli.Context) error {
	repo := c.Args().First()
	owner, name, err := parseRepo(repo)
	if err != nil {
		return err
	}

	in, err := readInput(c.String("in"))
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	sig, err := client.Sign(owner, name, in)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.String("out"), sig, 0664)
}
