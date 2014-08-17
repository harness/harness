package connection

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Connection struct {
	Driver string
	Source string
	DB     *gorm.DB
}

var database_url string

func NewConnection() *Connection {
	conn := &Connection{}

	// Setup Driver and Source from DATABASE_URL
	conn, err := conn.ParseUrl()
	if err != nil {
		panic(err)
	}

	return conn
}

func (c *Connection) Open() *Connection {
	db, err := gorm.Open(c.Driver, c.Source)
	if err != nil {
		panic(err)
	}

	c.DB = &db
	return c
}

func (c *Connection) Close() {
	c.DB.Close()
}

func (c *Connection) ParseUrl() (*Connection, error) {
	database_url = os.Getenv("DATABASE_URL")

	// If env varibale does not exist, set default
	if database_url == "" {
		// Get working directory
		working_directory, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		// Set sqlite3 as default driver
		c.Driver = "sqlite3"

		// If env test use memory driver, else use working directory
		if os.Getenv("DRONE") == "true" && os.Getenv("CI") == "true" {
			c.Source = ":memory:"
		} else {
			c.Source = fmt.Sprintf("%s/drone.sqlite", working_directory)
		}

		return c, nil
	}

	parsed_url, err := url.Parse(database_url)
	if err != nil {
		return nil, err
	}

	switch parsed_url.Scheme {
	case "sqlite3", "sqlite":
		c.Driver = "sqlite3"
		c.Source = parsed_url.RequestURI()
		return c, nil
	case "postgres", "postgresql":
		c.Driver = "postgres"
		c.Source, err = pq.ParseURL(database_url)
		return c, err
	case "mysql":
		c.Driver = "mysql"
		c.Source = BuildMysqlSource(parsed_url)
		return c, nil
	default:
		error_message := fmt.Sprintf("Unsupported driver: %s", parsed_url.Scheme)
		driver_error := errors.New(error_message)
		return nil, driver_error
	}
}

func BuildMysqlSource(parsed *url.URL) string {
	var source = ""

	if parsed.User.String() != "" {
		source = parsed.User.String() + "@"
	} else {
		panic("No user given")
	}

	if parsed.Host != "" {
		source = source + "tcp(" + parsed.Host + ")"
	} else {
		panic("No host given")
	}

	if parsed.Path != "" {
		source = source + parsed.Path
	} else {
		panic("No database given")
	}

	return source
}
