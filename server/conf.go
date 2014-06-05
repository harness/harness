package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// initialize the .drone directory and create a skeleton config
// file if one does not already exist.
func init() {
	// load the current user
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	// create the .drone home directory
	os.MkdirAll(filepath.Join(u.HomeDir, ".drone"), 0777)

	// check for the config file
	filename := filepath.Join(u.HomeDir, ".drone", "config.toml")
	if _, err := os.Stat(filename); err != nil {
		// if not exists, create
		ioutil.WriteFile(filename, []byte(defaultConfig), 0777)
	}

	// load the configuration file and parse
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
}

var defaultConfig = `
# Enables user self-registration. If false, the system administrator
# will need to manually add users to the system.
registration = true

[smtp]
host = ""
port = ""
from = ""
username = ""
password = ""

[bitbucket]
url = "https://bitbucket.org"
api = "https://bitbucket.org"
client = ""
secret = ""
enabled = false

[github]
url = "https://github.com"
api = "https://api.github.com"
client = ""
secret = ""
enabled = false

[githubenterprise]
url = ""
api = ""
client = ""
secret = ""
enabled = false

[gitlab]
url = ""
api = ""
client = ""
secret = ""
enabled = false

[stash]
url = ""
api = ""
client = ""
secret = ""
enabled = false
`
