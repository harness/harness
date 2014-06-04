package config

import (
	"fmt"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestRead(t *testing.T) {
	var data = `
scheme = "https"
host = "localhost"
port = 8080
open = true

[github]
url = "https://github.com"
api = "https://api.github.com"

[bitbucket]
url = "https://bitbucket.org"

[smtp]
host = "smtp.drone.io"
port = "443"
user = "brad"
from = "brad@drone.io"

`

	var conf Config
	if _, err := toml.Decode(data, &conf); err != nil {
		println(err.Error())
		return
	}

	fmt.Printf("%#v\n", conf)
}
