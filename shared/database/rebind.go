package database

import (
	"strconv"

	"github.com/russross/meddler"
)

// Rebind is a helper function that changes the sql
// bind type from ? to $ for postgres queries.
func Rebind(query string) string {
	if meddler.Default != meddler.PostgreSQL {
		return query
	}

	qb := []byte(query)
	// Add space enough for 5 params before we have to allocate
	rqb := make([]byte, 0, len(qb)+5)
	j := 1
	for _, b := range qb {
		switch b {
		case '?':
			rqb = append(rqb, '$')
			for _, b := range strconv.Itoa(j) {
				rqb = append(rqb, byte(b))
			}
			j++
		case '`':
			rqb = append(rqb, ' ')
		default:
			rqb = append(rqb, b)
		}
	}
	return string(rqb)
}
