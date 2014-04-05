package sqlite3

/*
#include <sqlite3.h>
#include <stdlib.h>
#include <string.h>

#ifdef __CYGWIN__
# include <errno.h>
#endif

#ifndef SQLITE_OPEN_READWRITE
# define SQLITE_OPEN_READWRITE 0
#endif

#ifndef SQLITE_OPEN_FULLMUTEX
# define SQLITE_OPEN_FULLMUTEX 0
#endif

static int
_sqlite3_open_v2(const char *filename, sqlite3 **ppDb, int flags, const char *zVfs) {
#ifdef SQLITE_OPEN_URI
  return sqlite3_open_v2(filename, ppDb, flags | SQLITE_OPEN_URI, zVfs);
#else
  return sqlite3_open_v2(filename, ppDb, flags, zVfs);
#endif
}

static int
_sqlite3_bind_text(sqlite3_stmt *stmt, int n, char *p, int np) {
  return sqlite3_bind_text(stmt, n, p, np, SQLITE_TRANSIENT);
}

static int
_sqlite3_bind_blob(sqlite3_stmt *stmt, int n, void *p, int np) {
  return sqlite3_bind_blob(stmt, n, p, np, SQLITE_TRANSIENT);
}

#include <stdio.h>
#include <stdint.h>

static long
_sqlite3_last_insert_rowid(sqlite3* db) {
  return (long) sqlite3_last_insert_rowid(db);
}

static long
_sqlite3_changes(sqlite3* db) {
  return (long) sqlite3_changes(db);
}

*/
import "C"
import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"time"
	"unsafe"
)

// Timestamp formats understood by both this module and SQLite.
// The first format in the slice will be used when saving time values
// into the database. When parsing a string from a timestamp or
// datetime column, the formats are tried in order.
var SQLiteTimestampFormats = []string{
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

func init() {
	sql.Register("sqlite3", &SQLiteDriver{})
}

// Driver struct.
type SQLiteDriver struct {
	Extensions  []string
	ConnectHook func(*SQLiteConn) error
}

// Conn struct.
type SQLiteConn struct {
	db *C.sqlite3
}

// Tx struct.
type SQLiteTx struct {
	c *SQLiteConn
}

// Stmt struct.
type SQLiteStmt struct {
	c      *SQLiteConn
	s      *C.sqlite3_stmt
	t      string
	closed bool
}

// Result struct.
type SQLiteResult struct {
	id      int64
	changes int64
}

// Rows struct.
type SQLiteRows struct {
	s        *SQLiteStmt
	nc       int
	cols     []string
	decltype []string
}

// Commit transaction.
func (tx *SQLiteTx) Commit() error {
	if err := tx.c.exec("COMMIT"); err != nil {
		return err
	}
	return nil
}

// Rollback transaction.
func (tx *SQLiteTx) Rollback() error {
	if err := tx.c.exec("ROLLBACK"); err != nil {
		return err
	}
	return nil
}

// AutoCommit return which currently auto commit or not.
func (c *SQLiteConn) AutoCommit() bool {
	return int(C.sqlite3_get_autocommit(c.db)) != 0
}

func (c *SQLiteConn) lastError() Error {
	return Error{Code: ErrNo(C.sqlite3_errcode(c.db)),
		err: C.GoString(C.sqlite3_errmsg(c.db)),
	}
}

// TODO: Execer & Queryer currently disabled
// https://github.com/mattn/go-sqlite3/issues/82
//// Implements Execer
//func (c *SQLiteConn) Exec(query string, args []driver.Value) (driver.Result, error) {
//	tx, err := c.Begin()
//	if err != nil {
//		return nil, err
//	}
//	for {
//		s, err := c.Prepare(query)
//		if err != nil {
//			tx.Rollback()
//			return nil, err
//		}
//		na := s.NumInput()
//		res, err := s.Exec(args[:na])
//		if err != nil && err != driver.ErrSkip {
//			tx.Rollback()
//			s.Close()
//			return nil, err
//		}
//		args = args[na:]
//		tail := s.(*SQLiteStmt).t
//		if tail == "" {
//			tx.Commit()
//			return res, nil
//		}
//		s.Close()
//		query = tail
//	}
//}
//
//// Implements Queryer
//func (c *SQLiteConn) Query(query string, args []driver.Value) (driver.Rows, error) {
//	tx, err := c.Begin()
//	if err != nil {
//		return nil, err
//	}
//	for {
//		s, err := c.Prepare(query)
//		if err != nil {
//			tx.Rollback()
//			return nil, err
//		}
//		na := s.NumInput()
//		rows, err := s.Query(args[:na])
//		if err != nil && err != driver.ErrSkip {
//			tx.Rollback()
//			s.Close()
//			return nil, err
//		}
//		args = args[na:]
//		tail := s.(*SQLiteStmt).t
//		if tail == "" {
//			tx.Commit()
//			return rows, nil
//		}
//		s.Close()
//		query = tail
//	}
//}

func (c *SQLiteConn) exec(cmd string) error {
	pcmd := C.CString(cmd)
	defer C.free(unsafe.Pointer(pcmd))
	rv := C.sqlite3_exec(c.db, pcmd, nil, nil, nil)
	if rv != C.SQLITE_OK {
		return c.lastError()
	}
	return nil
}

// Begin transaction.
func (c *SQLiteConn) Begin() (driver.Tx, error) {
	if err := c.exec("BEGIN"); err != nil {
		return nil, err
	}
	return &SQLiteTx{c}, nil
}

func errorString(err Error) string {
	return C.GoString(C.sqlite3_errstr(C.int(err.Code)))
}

// Open database and return a new connection.
// You can specify DSN string with URI filename.
//   test.db
//   file:test.db?cache=shared&mode=memory
//   :memory:
//   file::memory:
func (d *SQLiteDriver) Open(dsn string) (driver.Conn, error) {
	if C.sqlite3_threadsafe() == 0 {
		return nil, errors.New("sqlite library was not compiled for thread-safe operation")
	}

	var db *C.sqlite3
	name := C.CString(dsn)
	defer C.free(unsafe.Pointer(name))
	rv := C._sqlite3_open_v2(name, &db,
		C.SQLITE_OPEN_FULLMUTEX|
			C.SQLITE_OPEN_READWRITE|
			C.SQLITE_OPEN_CREATE,
		nil)
	if rv != 0 {
		return nil, Error{Code: ErrNo(rv)}
	}
	if db == nil {
		return nil, errors.New("sqlite succeeded without returning a database")
	}

	rv = C.sqlite3_busy_timeout(db, 5000)
	if rv != C.SQLITE_OK {
		return nil, Error{Code: ErrNo(rv)}
	}

	conn := &SQLiteConn{db}

	if len(d.Extensions) > 0 {
		rv = C.sqlite3_enable_load_extension(db, 1)
		if rv != C.SQLITE_OK {
			return nil, errors.New(C.GoString(C.sqlite3_errmsg(db)))
		}

		stmt, err := conn.Prepare("SELECT load_extension(?);")
		if err != nil {
			return nil, err
		}

		for _, extension := range d.Extensions {
			if _, err = stmt.Exec([]driver.Value{extension}); err != nil {
				return nil, err
			}
		}

		if err = stmt.Close(); err != nil {
			return nil, err
		}

		rv = C.sqlite3_enable_load_extension(db, 0)
		if rv != C.SQLITE_OK {
			return nil, errors.New(C.GoString(C.sqlite3_errmsg(db)))
		}
	}

	if d.ConnectHook != nil {
		if err := d.ConnectHook(conn); err != nil {
			return nil, err
		}
	}

	return conn, nil
}

// Close the connection.
func (c *SQLiteConn) Close() error {
	rv := C.sqlite3_close_v2(c.db)
	if rv != C.SQLITE_OK {
		return c.lastError()
	}
	c.db = nil
	return nil
}

// Prepare query string. Return a new statement.
func (c *SQLiteConn) Prepare(query string) (driver.Stmt, error) {
	pquery := C.CString(query)
	defer C.free(unsafe.Pointer(pquery))
	var s *C.sqlite3_stmt
	var tail *C.char
	rv := C.sqlite3_prepare_v2(c.db, pquery, -1, &s, &tail)
	if rv != C.SQLITE_OK {
		return nil, c.lastError()
	}
	var t string
	if tail != nil && C.strlen(tail) > 0 {
		t = strings.TrimSpace(C.GoString(tail))
	}
	return &SQLiteStmt{c: c, s: s, t: t}, nil
}

// Close the statement.
func (s *SQLiteStmt) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.c == nil || s.c.db == nil {
		return errors.New("sqlite statement with already closed database connection")
	}
	rv := C.sqlite3_finalize(s.s)
	if rv != C.SQLITE_OK {
		return s.c.lastError()
	}
	return nil
}

// Return a number of parameters.
func (s *SQLiteStmt) NumInput() int {
	return int(C.sqlite3_bind_parameter_count(s.s))
}

func (s *SQLiteStmt) bind(args []driver.Value) error {
	rv := C.sqlite3_reset(s.s)
	if rv != C.SQLITE_ROW && rv != C.SQLITE_OK && rv != C.SQLITE_DONE {
		return s.c.lastError()
	}

	for i, v := range args {
		n := C.int(i + 1)
		switch v := v.(type) {
		case nil:
			rv = C.sqlite3_bind_null(s.s, n)
		case string:
			if len(v) == 0 {
				b := []byte{0}
				rv = C._sqlite3_bind_text(s.s, n, (*C.char)(unsafe.Pointer(&b[0])), C.int(0))
			} else {
				b := []byte(v)
				rv = C._sqlite3_bind_text(s.s, n, (*C.char)(unsafe.Pointer(&b[0])), C.int(len(b)))
			}
		case int:
			rv = C.sqlite3_bind_int64(s.s, n, C.sqlite3_int64(v))
		case int32:
			rv = C.sqlite3_bind_int(s.s, n, C.int(v))
		case int64:
			rv = C.sqlite3_bind_int64(s.s, n, C.sqlite3_int64(v))
		case byte:
			rv = C.sqlite3_bind_int(s.s, n, C.int(v))
		case bool:
			if bool(v) {
				rv = C.sqlite3_bind_int(s.s, n, 1)
			} else {
				rv = C.sqlite3_bind_int(s.s, n, 0)
			}
		case float32:
			rv = C.sqlite3_bind_double(s.s, n, C.double(v))
		case float64:
			rv = C.sqlite3_bind_double(s.s, n, C.double(v))
		case []byte:
			var p *byte
			if len(v) > 0 {
				p = &v[0]
			}
			rv = C._sqlite3_bind_blob(s.s, n, unsafe.Pointer(p), C.int(len(v)))
		case time.Time:
			b := []byte(v.UTC().Format(SQLiteTimestampFormats[0]))
			rv = C._sqlite3_bind_text(s.s, n, (*C.char)(unsafe.Pointer(&b[0])), C.int(len(b)))
		}
		if rv != C.SQLITE_OK {
			return s.c.lastError()
		}
	}
	return nil
}

// Query the statement with arguments. Return records.
func (s *SQLiteStmt) Query(args []driver.Value) (driver.Rows, error) {
	if err := s.bind(args); err != nil {
		return nil, err
	}
	return &SQLiteRows{s, int(C.sqlite3_column_count(s.s)), nil, nil}, nil
}

// Return last inserted ID.
func (r *SQLiteResult) LastInsertId() (int64, error) {
	return r.id, nil
}

// Return how many rows affected.
func (r *SQLiteResult) RowsAffected() (int64, error) {
	return r.changes, nil
}

// Execute the statement with arguments. Return result object.
func (s *SQLiteStmt) Exec(args []driver.Value) (driver.Result, error) {
	if err := s.bind(args); err != nil {
		return nil, err
	}
	rv := C.sqlite3_step(s.s)
	if rv != C.SQLITE_ROW && rv != C.SQLITE_OK && rv != C.SQLITE_DONE {
		return nil, s.c.lastError()
	}

	res := &SQLiteResult{
		int64(C._sqlite3_last_insert_rowid(s.c.db)),
		int64(C._sqlite3_changes(s.c.db)),
	}
	return res, nil
}

// Close the rows.
func (rc *SQLiteRows) Close() error {
	if rc.s.closed {
		return nil
	}
	rv := C.sqlite3_reset(rc.s.s)
	if rv != C.SQLITE_OK {
		return rc.s.c.lastError()
	}
	return nil
}

// Return column names.
func (rc *SQLiteRows) Columns() []string {
	if rc.nc != len(rc.cols) {
		rc.cols = make([]string, rc.nc)
		for i := 0; i < rc.nc; i++ {
			rc.cols[i] = C.GoString(C.sqlite3_column_name(rc.s.s, C.int(i)))
		}
	}
	return rc.cols
}

// Move cursor to next.
func (rc *SQLiteRows) Next(dest []driver.Value) error {
	rv := C.sqlite3_step(rc.s.s)
	if rv == C.SQLITE_DONE {
		return io.EOF
	}
	if rv != C.SQLITE_ROW {
		rv = C.sqlite3_reset(rc.s.s)
		if rv != C.SQLITE_OK {
			return rc.s.c.lastError()
		}
		return nil
	}

	if rc.decltype == nil {
		rc.decltype = make([]string, rc.nc)
		for i := 0; i < rc.nc; i++ {
			rc.decltype[i] = strings.ToLower(C.GoString(C.sqlite3_column_decltype(rc.s.s, C.int(i))))
		}
	}

	for i := range dest {
		switch C.sqlite3_column_type(rc.s.s, C.int(i)) {
		case C.SQLITE_INTEGER:
			val := int64(C.sqlite3_column_int64(rc.s.s, C.int(i)))
			switch rc.decltype[i] {
			case "timestamp", "datetime":
				dest[i] = time.Unix(val, 0)
			case "boolean":
				dest[i] = val > 0
			default:
				dest[i] = val
			}
		case C.SQLITE_FLOAT:
			dest[i] = float64(C.sqlite3_column_double(rc.s.s, C.int(i)))
		case C.SQLITE_BLOB:
			p := C.sqlite3_column_blob(rc.s.s, C.int(i))
			if p == nil {
				dest[i] = nil
				continue
			}
			n := int(C.sqlite3_column_bytes(rc.s.s, C.int(i)))
			switch dest[i].(type) {
			case sql.RawBytes:
				dest[i] = (*[1 << 30]byte)(unsafe.Pointer(p))[0:n]
			default:
				slice := make([]byte, n)
				copy(slice[:], (*[1 << 30]byte)(unsafe.Pointer(p))[0:n])
				dest[i] = slice
			}
		case C.SQLITE_NULL:
			dest[i] = nil
		case C.SQLITE_TEXT:
			var err error
			s := C.GoString((*C.char)(unsafe.Pointer(C.sqlite3_column_text(rc.s.s, C.int(i)))))

			switch rc.decltype[i] {
			case "timestamp", "datetime":
				for _, format := range SQLiteTimestampFormats {
					if dest[i], err = time.Parse(format, s); err == nil {
						break
					}
				}
				if err != nil {
					// The column is a time value, so return the zero time on parse failure.
					dest[i] = time.Time{}
				}
			default:
				dest[i] = []byte(s)
			}

		}
	}
	return nil
}
