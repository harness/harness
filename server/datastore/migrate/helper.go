package migrate

import (
	"strconv"
	"strings"

	"github.com/russross/meddler"
)

// transform is a helper function that transforms sql
// statements to work with multiple database types.
func transform(stmt string) string {
	switch meddler.Default {
	case meddler.MySQL:
		stmt = strings.Replace(stmt, "AUTOINCREMENT", "AUTO_INCREMENT", -1)
	case meddler.PostgreSQL:
		stmt = strings.Replace(stmt, "INTEGER PRIMARY KEY AUTOINCREMENT", "SERIAL PRIMARY KEY", -1)
		stmt = strings.Replace(stmt, "BLOB", "BYTEA", -1)
	}
	return stmt
}

// rebind is a helper function that changes the sql
// bind type from ? to $ for postgres queries.
func rebind(query string) string {
	if meddler.Default != meddler.PostgreSQL {
		return query
	}

	qb := []byte(query)
	// Add space enough for 10 params before we have to allocate
	rqb := make([]byte, 0, len(qb)+10)
	j := 1
	for _, b := range qb {
		if b == '?' {
			rqb = append(rqb, '$')
			for _, b := range strconv.Itoa(j) {
				rqb = append(rqb, byte(b))
			}
			j++
		} else {
			rqb = append(rqb, b)
		}
	}
	return string(rqb)
}
