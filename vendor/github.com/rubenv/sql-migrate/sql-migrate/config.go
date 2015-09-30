package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/rubenv/sql-migrate"
	"gopkg.in/gorp.v1"
	"gopkg.in/yaml.v1"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var dialects = map[string]gorp.Dialect{
	"sqlite3":  gorp.SqliteDialect{},
	"postgres": gorp.PostgresDialect{},
	"mysql":    gorp.MySQLDialect{"InnoDB", "UTF8"},
}

var ConfigFile string
var ConfigEnvironment string

func ConfigFlags(f *flag.FlagSet) {
	f.StringVar(&ConfigFile, "config", "dbconfig.yml", "Configuration file to use.")
	f.StringVar(&ConfigEnvironment, "env", "development", "Environment to use.")
}

type Environment struct {
	Dialect    string `yaml:"dialect"`
	DataSource string `yaml:"datasource"`
	Dir        string `yaml:"dir"`
	TableName  string `yaml:"table"`
	SchemaName string `yaml:"schema"`
}

func ReadConfig() (map[string]*Environment, error) {
	file, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}

	config := make(map[string]*Environment)
	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func GetEnvironment() (*Environment, error) {
	config, err := ReadConfig()
	if err != nil {
		return nil, err
	}

	env := config[ConfigEnvironment]
	if env == nil {
		return nil, errors.New("No environment: " + ConfigEnvironment)
	}

	if env.Dialect == "" {
		return nil, errors.New("No dialect specified")
	}

	if env.DataSource == "" {
		return nil, errors.New("No data source specified")
	}

	if env.Dir == "" {
		env.Dir = "migrations"
	}

	if env.TableName != "" {
		migrate.SetTable(env.TableName)
	}

	if env.SchemaName != "" {
		migrate.SetSchema(env.SchemaName)
	}

	return env, nil
}

func GetConnection(env *Environment) (*sql.DB, string, error) {
	db, err := sql.Open(env.Dialect, env.DataSource)
	if err != nil {
		return nil, "", fmt.Errorf("Cannot connect to database: %s", err)
	}

	// Make sure we only accept dialects that were compiled in.
	_, exists := dialects[env.Dialect]
	if !exists {
		return nil, "", fmt.Errorf("Unsupported dialect: %s", env.Dialect)
	}

	return db, env.Dialect, nil
}
