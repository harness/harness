package connection

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Connection struct {
	Driver string
	Source string
	DB     *gorm.DB
}

func NewConnection() *Connection {
	return &Connection{}
}

func (c *Connection) Open() *Connection {
	wd, _ := os.Getwd()

	if os.Getenv("DRONE_DRIVER") == "" {
		os.Setenv("DRONE_DRIVER", "sqlite3")
		log.Println("WARNING: env variable DRONE_DRIVER is missing, use:", os.Getenv("DRONE_DRIVER"))
	}

	if os.Getenv("DRONE_DATASOURCE") == "" {
		os.Setenv("DRONE_DATASOURCE", wd+"/drone.sqlite")
		log.Println("WARNING: env variable DRONE_DATASOURCE is missing, use:", os.Getenv("DRONE_DATASOURCE"))
	}

	db, err := gorm.Open(os.Getenv("DRONE_DRIVER"), os.Getenv("DRONE_DATASOURCE"))
	if err != nil {
		panic(err)
	}

	c.DB = &db
	return c
}

func (c *Connection) Close() {
	c.DB.Close()
}
