package datastore

import (
	"strconv"
	"strings"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
	"math"
)

// rebind is a helper function that changes the sql
// bind type from ? to $ for postgres queries.
func rebind(query string) string {
	if meddler.Default != meddler.PostgreSQL {
		return query
	}

	qb := []byte(query)
	// Add space enough for 5 params before we have to allocate
	rqb := make([]byte, 0, len(qb) + 5)
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
func toList(listof []*model.RepoLite) ([]string, [][]interface{}) {
	var total = len(listof)
	var limit = 999
	var pagesCount = int(math.Ceil(float64(total) / float64(limit)))

	var qs = make([]string, pagesCount, pagesCount)
	var in = make([][]interface{}, pagesCount, pagesCount)

	for page := 0; page < int(pagesCount); page++ {
		var start = page * limit
		var end = (page * limit) + limit
		if end > total{
			end = total
		}
		var size = end - start
		var _qs = make([]string, size, size)
		var _in = make([]interface{}, size, size)
		for i, repo := range listof[start:end] {
			if meddler.Default == meddler.PostgreSQL {
				_qs[i] = "$" + strconv.Itoa(i + 1)
			}else{
				_qs[i] = "?"
			}
			_in[i] = repo.FullName
		}
		qs = append(qs, strings.Join(_qs, ","))
		in = append(in, _in)
	}
	return qs, in
}