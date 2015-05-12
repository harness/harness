package build

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
)

// createUID is a helper function that will
// create a random, unique identifier.
func createUID() string {
	c := sha1.New()
	r := createRandom()
	io.WriteString(c, string(r))
	s := fmt.Sprintf("%x", c.Sum(nil))
	return "drone-" + s[0:10]
}

// createRandom creates a random block of bytes
// that we can use to generate unique identifiers.
func createRandom() []byte {
	k := make([]byte, sha1.BlockSize)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

// list of service aliases and their full, canonical names
var defaultServices = map[string]string{
	"cassandra":     "relateiq/cassandra:latest",
	"couchdb":       "bradrydzewski/couchdb:1.5",
	"elasticsearch": "bradrydzewski/elasticsearch:0.90",
	"memcached":     "bradrydzewski/memcached",
	"mongodb":       "bradrydzewski/mongodb:2.4",
	"mysql":         "bradrydzewski/mysql:5.5",
	"neo4j":         "bradrydzewski/neo4j:1.9",
	"postgres":      "bradrydzewski/postgres:9.1",
	"redis":         "bradrydzewski/redis:2.8",
	"rabbitmq":      "bradrydzewski/rabbitmq:3.2",
	"riak":          "guillermo/riak:latest",
	"zookeeper":     "jplock/zookeeper:3.4.5",
}

// parseImageName parses a Docker image name, in the format owner/name:tag,
// and returns each segment.
//
// If the owner is blank, it is assumed to be an official drone image,
// and will be prefixed with the appropriate owner name.
//
// If the tag is empty, it is assumed to be the latest version.
func parseImageName(image string) (owner, name, tag string) {
	owner = "bradrydzewski" // this will eventually change to drone
	name = image
	tag = "latest"

	// first we check to see if the image name is an alias
	// for a known service.
	//
	// TODO I'm not a huge fan of this code here. Maybe it
	//      should get handled when the yaml is parsed, and
	//      convert the image and service names in the yaml
	//      to fully qualified names?
	if cname, ok := defaultServices[image]; ok {
		name = cname
	}

	parts := strings.Split(name, "/")
	if len := len(parts); len == 3 {
		owner = fmt.Sprintf("%s/%s", parts[0], parts[1])
		name = parts[2]
	} else if len == 2 {
		owner = parts[0]
		name = parts[1]
	}

	parts = strings.Split(name, ":")
	if len(parts) == 2 {
		name = parts[0]
		tag = parts[1]
	}

	return
}
