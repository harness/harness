package database

import (
	"strconv"

	"github.com/russross/meddler"
)

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
