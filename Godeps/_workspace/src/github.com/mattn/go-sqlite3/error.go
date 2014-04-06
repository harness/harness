package sqlite3

import "C"

type ErrNo int

type Error struct {
	Code ErrNo  /* The error code returned by SQLite */
	err  string /* The error string returned by sqlite3_errmsg(),
	this usually contains more specific details. */
}

// result codes from http://www.sqlite.org/c3ref/c_abort.html
var (
	ErrError      error = ErrNo(1)  /* SQL error or missing database */
	ErrInternal   error = ErrNo(2)  /* Internal logic error in SQLite */
	ErrPerm       error = ErrNo(3)  /* Access permission denied */
	ErrAbort      error = ErrNo(4)  /* Callback routine requested an abort */
	ErrBusy       error = ErrNo(5)  /* The database file is locked */
	ErrLocked     error = ErrNo(6)  /* A table in the database is locked */
	ErrNomem      error = ErrNo(7)  /* A malloc() failed */
	ErrReadonly   error = ErrNo(8)  /* Attempt to write a readonly database */
	ErrInterrupt  error = ErrNo(9)  /* Operation terminated by sqlite3_interrupt() */
	ErrIoErr      error = ErrNo(10) /* Some kind of disk I/O error occurred */
	ErrCorrupt    error = ErrNo(11) /* The database disk image is malformed */
	ErrNotFound   error = ErrNo(12) /* Unknown opcode in sqlite3_file_control() */
	ErrFull       error = ErrNo(13) /* Insertion failed because database is full */
	ErrCantOpen   error = ErrNo(14) /* Unable to open the database file */
	ErrProtocol   error = ErrNo(15) /* Database lock protocol error */
	ErrEmpty      error = ErrNo(16) /* Database is empty */
	ErrSchema     error = ErrNo(17) /* The database schema changed */
	ErrTooBig     error = ErrNo(18) /* String or BLOB exceeds size limit */
	ErrConstraint error = ErrNo(19) /* Abort due to constraint violation */
	ErrMismatch   error = ErrNo(20) /* Data type mismatch */
	ErrMisuse     error = ErrNo(21) /* Library used incorrectly */
	ErrNoLFS      error = ErrNo(22) /* Uses OS features not supported on host */
	ErrAuth       error = ErrNo(23) /* Authorization denied */
	ErrFormat     error = ErrNo(24) /* Auxiliary database format error */
	ErrRange      error = ErrNo(25) /* 2nd parameter to sqlite3_bind out of range */
	ErrNotADB     error = ErrNo(26) /* File opened that is not a database file */
	ErrNotice     error = ErrNo(27) /* Notifications from sqlite3_log() */
	ErrWarning    error = ErrNo(28) /* Warnings from sqlite3_log() */
)

func (err ErrNo) Error() string {
	return Error{Code: err}.Error()
}

func (err Error) Error() string {
	if err.err != "" {
		return err.err
	}
	return errorString(err)
}
