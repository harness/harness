package datastore

import (
	"strconv"
	"strings"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

// rebind is a helper function that changes the sql
// bind type from ? to $ for postgres queries.
func rebind(query string) string {
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

// helper function that converts a simple repsitory list
// to a sql IN statment.
func toList(listof []*model.RepoLite) (string, []interface{}) {
	var size = len(listof)
	switch {
	case meddler.Default == meddler.SQLite && size > 999:
		size = 999
		listof = listof[:999]
	case size > 15000:
		size = 15000
		listof = listof[:15000]
	}
	var qs = make([]string, size, size)
	var in = make([]interface{}, size, size)
	for i, repo := range listof {
		qs[i] = "?"
		in[i] = repo.FullName
	}
	return strings.Join(qs, ","), in
}

// helper function that converts a simple repository list
// to a sql IN statement compatible with postgres.
func toListPostgres(listof []*model.RepoLite) (string, []interface{}) {
	var size = len(listof)
	if size > 15000 {
		size = 15000
		listof = listof[:15000]
	}
	var qs = make([]string, size, size)
	var in = make([]interface{}, size, size)
	for i, repo := range listof {
		qs[i] = "$" + strconv.Itoa(i+1)
		in[i] = repo.FullName
	}
	return strings.Join(qs, ","), in
}
