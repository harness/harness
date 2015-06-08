package build

type image struct {
	// default ports the service will run on.
	// for example, 3306 for mysql. Note that a service
	// may expose multiple prots, for example, Riak
	// exposes 8087 and 8089.
	Ports []string

	// tag of the docker image to pull in order
	// to run this service.
	Tag string

	// display name of the image type
	Name string
}

// List of 3rd party services (database, queue, etc) that
// are known to work with this Build utility.
var services = map[string]*image{

	// neo4j
	"neo4j": {
		Ports: []string{"7474"},
		Tag:   "bradrydzewski/neo4j:1.9",
		Name:  "neo4j",
	},
	"neo4j:1.9": {
		Ports: []string{"7474"},
		Tag:   "bradrydzewski/neo4j:1.9",
		Name:  "neo4j",
	},

	// elasticsearch servers
	"elasticsearch": {
		Ports: []string{"9200"},
		Tag:   "bradrydzewski/elasticsearch:0.90",
		Name:  "elasticsearch",
	},
	"elasticsearch:0.20": {
		Ports: []string{"9200"},
		Tag:   "bradrydzewski/elasticsearch:0.20",
		Name:  "elasticsearch",
	},
	"elasticsearch:0.90": {
		Ports: []string{"9200"},
		Tag:   "bradrydzewski/elasticsearch:0.90",
		Name:  "elasticsearch",
	},

	// redis servers
	"redis": {
		Ports: []string{"6379"},
		Tag:   "bradrydzewski/redis:2.8",
		Name:  "redis",
	},
	"redis:2.8": {
		Ports: []string{"6379"},
		Tag:   "bradrydzewski/redis:2.8",
		Name:  "redis",
	},
	"redis:2.6": {
		Ports: []string{"6379"},
		Tag:   "bradrydzewski/redis:2.6",
		Name:  "redis",
	},

	// mysql servers
	"mysql": {
		Tag:   "bradrydzewski/mysql:5.5",
		Ports: []string{"3306"},
		Name:  "mysql",
	},
	"mysql:5.5": {
		Tag:   "bradrydzewski/mysql:5.5",
		Ports: []string{"3306"},
		Name:  "mysql",
	},

	// memcached
	"memcached": {
		Ports: []string{"11211"},
		Tag:   "bradrydzewski/memcached",
		Name:  "memcached",
	},

	// mongodb
	"mongodb": {
		Ports: []string{"27017"},
		Tag:   "bradrydzewski/mongodb:2.4",
		Name:  "mongodb",
	},
	"mongodb:2.4": {
		Ports: []string{"27017"},
		Tag:   "bradrydzewski/mongodb:2.4",
		Name:  "mongodb",
	},
	"mongodb:2.2": {
		Ports: []string{"27017"},
		Tag:   "bradrydzewski/mongodb:2.2",
		Name:  "mongodb",
	},

	// postgres
	"postgres": {
		Ports: []string{"5432"},
		Tag:   "bradrydzewski/postgres:9.1",
		Name:  "postgres",
	},
	"postgres:9.1": {
		Ports: []string{"5432"},
		Tag:   "bradrydzewski/postgres:9.1",
		Name:  "postgres",
	},

	// couchdb
	"couchdb": {
		Ports: []string{"5984"},
		Tag:   "bradrydzewski/couchdb:1.5",
		Name:  "couchdb",
	},
	"couchdb:1.0": {
		Ports: []string{"5984"},
		Tag:   "bradrydzewski/couchdb:1.0",
		Name:  "couchdb",
	},
	"couchdb:1.4": {
		Ports: []string{"5984"},
		Tag:   "bradrydzewski/couchdb:1.4",
		Name:  "couchdb",
	},
	"couchdb:1.5": {
		Ports: []string{"5984"},
		Tag:   "bradrydzewski/couchdb:1.5",
		Name:  "couchdb",
	},

	// rabbitmq
	"rabbitmq": {
		Ports: []string{"5672", "15672"},
		Tag:   "bradrydzewski/rabbitmq:3.2",
		Name:  "rabbitmq",
	},
	"rabbitmq:3.2": {
		Ports: []string{"5672", "15672"},
		Tag:   "bradrydzewski/rabbitmq:3.2",
		Name:  "rabbitmq",
	},

	// experimental images from 3rd parties

	"zookeeper": {
		Ports: []string{"2181"},
		Tag:   "jplock/zookeeper:3.4.5",
		Name:  "zookeeper",
	},

	// cassandra
	"cassandra": {
		Ports: []string{"9042", "7000", "7001", "7199", "9160", "49183"},
		Tag:   "relateiq/cassandra",
		Name:  "cassandra",
	},

	// riak - TESTED
	"riak": {
		Ports: []string{"8087", "8098"},
		Tag:   "guillermo/riak",
		Name:  "riak",
	},
}

// List of official Drone build images.
var builders = map[string]*image{

	// Clojure build images
	"lein": {Tag: "bradrydzewski/lein"},

	// Dart build images
	"dart":        {Tag: "bradrydzewski/dart:stable"},
	"dart_stable": {Tag: "bradrydzewski/dart:stable"},
	"dart_dev":    {Tag: "bradrydzewski/dart:dev"},

	// Erlang build images
	"erlang":       {Tag: "bradrydzewski/erlang:R16B02"},
	"erlangR16B":   {Tag: "bradrydzewski/erlang:R16B"},
	"erlangR16B02": {Tag: "bradrydzewski/erlang:R16B02"},
	"erlangR16B01": {Tag: "bradrydzewski/erlang:R16B01"},

	// GCC build images
	"gcc":    {Tag: "bradrydzewski/gcc:4.6"},
	"gcc4.6": {Tag: "bradrydzewski/gcc:4.6"},
	"gcc4.8": {Tag: "bradrydzewski/gcc:4.8"},

	// Golang build images
	"go":    {Tag: "bradrydzewski/go:1.3"},
	"go1":   {Tag: "bradrydzewski/go:1.0"},
	"go1.1": {Tag: "bradrydzewski/go:1.1"},
	"go1.2": {Tag: "bradrydzewski/go:1.2"},
	"go1.3": {Tag: "bradrydzewski/go:1.3"},
	"go1.4": {Tag: "bradrydzewski/go:1.4"},

	// Haskell build images
	"haskell":    {Tag: "bradrydzewski/haskell:7.4"},
	"haskell7.4": {Tag: "bradrydzewski/haskell:7.4"},

	// Java build images
	"java":       {Tag: "bradrydzewski/java:openjdk7"},
	"openjdk6":   {Tag: "bradrydzewski/java:openjdk6"},
	"openjdk7":   {Tag: "bradrydzewski/java:openjdk7"},
	"oraclejdk7": {Tag: "bradrydzewski/java:oraclejdk7"},
	"oraclejdk8": {Tag: "bradrydzewski/java:oraclejdk8"},

	// Node build images
	"node":     {Tag: "bradrydzewski/node:0.10"},
	"node0.10": {Tag: "bradrydzewski/node:0.10"},
	"node0.8":  {Tag: "bradrydzewski/node:0.8"},

	// PHP build images
	"php":    {Tag: "bradrydzewski/php:5.5"},
	"php5.5": {Tag: "bradrydzewski/php:5.5"},
	"php5.4": {Tag: "bradrydzewski/php:5.4"},

	// Python build images
	"python":    {Tag: "bradrydzewski/python:2.7"},
	"python2.7": {Tag: "bradrydzewski/python:2.7"},
	"python3.2": {Tag: "bradrydzewski/python:3.2"},
	"python3.3": {Tag: "bradrydzewski/python:3.3"},
	"pypy":      {Tag: "bradrydzewski/python:pypy"},

	// Ruby build images
	"ruby":      {Tag: "bradrydzewski/ruby:2.0.0"},
	"ruby2.0.0": {Tag: "bradrydzewski/ruby:2.0.0"},
	"ruby1.9.3": {Tag: "bradrydzewski/ruby:1.9.3"},

	// Scala build images
	"scala":       {Tag: "bradrydzewski/scala:2.10.3"},
	"scala2.10.3": {Tag: "bradrydzewski/scala:2.10.3"},
	"scala2.9.3":  {Tag: "bradrydzewski/scala:2.9.3"},
}
