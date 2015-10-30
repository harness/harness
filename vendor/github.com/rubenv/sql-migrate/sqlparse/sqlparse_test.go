package sqlparse

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SqlParseSuite struct {
}

var _ = Suite(&SqlParseSuite{})

func (s *SqlParseSuite) TestSemicolons(c *C) {
	type testData struct {
		line   string
		result bool
	}

	tests := []testData{
		{
			line:   "END;",
			result: true,
		},
		{
			line:   "END; -- comment",
			result: true,
		},
		{
			line:   "END   ; -- comment",
			result: true,
		},
		{
			line:   "END -- comment",
			result: false,
		},
		{
			line:   "END -- comment ;",
			result: false,
		},
		{
			line:   "END \" ; \" -- comment",
			result: false,
		},
	}

	for _, test := range tests {
		r := endsWithSemicolon(test.line)
		c.Assert(r, Equals, test.result)
	}
}

func (s *SqlParseSuite) TestSplitStatements(c *C) {
	type testData struct {
		sql       string
		direction bool
		count     int
	}

	tests := []testData{
		{
			sql:       functxt,
			direction: true,
			count:     2,
		},
		{
			sql:       functxt,
			direction: false,
			count:     2,
		},
		{
			sql:       multitxt,
			direction: true,
			count:     2,
		},
		{
			sql:       multitxt,
			direction: false,
			count:     2,
		},
	}

	for _, test := range tests {
		stmts, err := SplitSQLStatements(strings.NewReader(test.sql), test.direction)
		c.Assert(err, IsNil)
		c.Assert(stmts, HasLen, test.count)
	}
}

var functxt = `-- +migrate Up
CREATE TABLE IF NOT EXISTS histories (
  id                BIGSERIAL  PRIMARY KEY,
  current_value     varchar(2000) NOT NULL,
  created_at      timestamp with time zone  NOT NULL
);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
BEGIN
  FOR create_query IN SELECT
      'CREATE TABLE IF NOT EXISTS histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( created_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND created_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( histories );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
  END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;
-- +migrate StatementEnd

-- +migrate Down
drop function histories_partition_creation(DATE, DATE);
drop TABLE histories;
`

// test multiple up/down transitions in a single script
var multitxt = `-- +migrate Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +migrate Down
DROP TABLE post;

-- +migrate Up
CREATE TABLE fancier_post (
    id int NOT NULL,
    title text,
    body text,
    created_on timestamp without time zone,
    PRIMARY KEY(id)
);

-- +migrate Down
DROP TABLE fancier_post;
`
