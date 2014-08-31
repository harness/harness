package connection

import (
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
