// +build !windows

package sqlite3

/*
#cgo CFLAGS: -I.
#cgo linux LDFLAGS: -ldl
#cgo CFLAGS: -DSQLITE_ENABLE_RTREE
*/
import "C"
