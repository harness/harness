package main

import (
	_ "github.com/drone/drone/common"
	"github.com/drone/drone/datastore"
	"github.com/drone/drone/datastore/bolt"
)

var (
	revision string
	version  string
)

var ds datastore.Datastore

func main() {
	ds, _ = bolt.New("drone.toml")
	println(revision)
	println(version)
}
