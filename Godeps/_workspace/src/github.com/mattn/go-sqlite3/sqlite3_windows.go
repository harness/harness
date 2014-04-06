package sqlite3

/*
#cgo CFLAGS: -I. -fno-stack-check -fno-stack-protector -mno-stack-arg-probe
#cgo LDFLAGS: -lmingwex -lmingw32
#cgo CFLAGS: -DSQLITE_ENABLE_RTREE
*/
import "C"
