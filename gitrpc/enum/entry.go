package enum

// EntryMode is the unix file mode of a tree entry.
type EntryMode int

// There are only a few file modes in Git. They look like unix file modes, but
// they can only be one of these.
const (
	EntryTree    EntryMode = 0040000
	EntryBlob    EntryMode = 0100644
	EntryExec    EntryMode = 0100755
	EntrySymlink EntryMode = 0120000
	EntryCommit  EntryMode = 0160000
)
