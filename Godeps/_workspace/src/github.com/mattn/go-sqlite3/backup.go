package sqlite3

/*
#include <sqlite3.h>
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

type Backup struct {
	b *C.sqlite3_backup
}

func (c *SQLiteConn) Backup(dest string, conn *SQLiteConn, src string) (*Backup, error) {
	destptr := C.CString(dest)
	defer C.free(unsafe.Pointer(destptr))
	srcptr := C.CString(src)
	defer C.free(unsafe.Pointer(srcptr))

	if b := C.sqlite3_backup_init(c.db, destptr, conn.db, srcptr); b != nil {
		return &Backup{b: b}, nil
	}
	return nil, c.lastError()
}

func (b *Backup) Step(p int) error {
	return Error{Code: ErrNo(C.sqlite3_backup_step(b.b, C.int(p)))}
}

func (b *Backup) Remaining() int {
	return int(C.sqlite3_backup_remaining(b.b))
}

func (b *Backup) PageCount() int {
	return int(C.sqlite3_backup_pagecount(b.b))
}

func (b *Backup) Finish() error {
	return Error{Code: ErrNo(C.sqlite3_backup_finish(b.b))}
}
