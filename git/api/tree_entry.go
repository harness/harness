package api

import (
	"strconv"
)

// EntryMode the type of the object in the git tree
type EntryMode int

// There are only a few file modes in Git. They look like unix file modes, but they can only be
// one of these.
const (
	EntryModeBlob    EntryMode = 0o100644
	EntryModeExec    EntryMode = 0o100755
	EntryModeSymlink EntryMode = 0o120000
	EntryModeCommit  EntryMode = 0o160000
	EntryModeTree    EntryMode = 0o040000
)

// String converts an EntryMode to a string
func (e EntryMode) String() string {
	return strconv.FormatInt(int64(e), 8)
}

// ToEntryMode converts a string to an EntryMode
func ToEntryMode(value string) EntryMode {
	v, _ := strconv.ParseInt(value, 8, 32)
	return EntryMode(v)
}

// TreeEntry the leaf in the git tree
type TreeEntry struct {
	ID SHA

	ptree *Tree

	entryMode EntryMode
	name      string

	size     int64
	sized    bool
	fullName string
}

// Name returns the name of the entry
func (te *TreeEntry) Name() string {
	if te.fullName != "" {
		return te.fullName
	}
	return te.name
}

// Mode returns the mode of the entry
func (te *TreeEntry) Mode() EntryMode {
	return te.entryMode
}

// IsSubModule if the entry is a sub module
func (te *TreeEntry) IsSubModule() bool {
	return te.entryMode == EntryModeCommit
}

// IsDir if the entry is a sub dir
func (te *TreeEntry) IsDir() bool {
	return te.entryMode == EntryModeTree
}

// IsLink if the entry is a symlink
func (te *TreeEntry) IsLink() bool {
	return te.entryMode == EntryModeSymlink
}

// IsRegular if the entry is a regular file
func (te *TreeEntry) IsRegular() bool {
	return te.entryMode == EntryModeBlob
}

// IsExecutable if the entry is an executable file (not necessarily binary)
func (te *TreeEntry) IsExecutable() bool {
	return te.entryMode == EntryModeExec
}
