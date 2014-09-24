package rice

import (
	"archive/zip"
	"bitbucket.org/kardianos/osext"
	"github.com/daaku/go.zipexe"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// appendedBox defines an appended box
type appendedBox struct {
	Name  string                   // box name
	Files map[string]*appendedFile // appended files (*zip.File) by full path
}

type appendedFile struct {
	zipFile  *zip.File
	dir      bool
	dirInfo  *appendedDirInfo
	children []*appendedFile
}

// appendedBoxes is a public register of appendes boxes
var appendedBoxes = make(map[string]*appendedBox)

func init() {
	// find if exec is appended
	thisFile, err := osext.Executable()
	if err != nil {
		return // not apended
	}
	rd, err := zipexe.Open(thisFile)
	if err != nil {
		return // not apended
	}

	for _, f := range rd.File {
		// get box and file name from f.Name
		fileParts := strings.SplitN(strings.TrimLeft(f.Name, "/"), "/", 2)
		boxName := fileParts[0]
		var fileName string
		if len(fileParts) > 1 {
			fileName = fileParts[1]
		}

		// find box or create new one if doesn't exist
		box := appendedBoxes[boxName]
		if box == nil {
			box = &appendedBox{
				Name:  boxName,
				Files: make(map[string]*appendedFile),
			}
			appendedBoxes[boxName] = box
		}

		// create and add file to box
		af := &appendedFile{
			zipFile: f,
		}
		if f.Comment == "dir" {
			af.dir = true
			af.dirInfo = &appendedDirInfo{
				name: filepath.Base(af.zipFile.Name),
				//++ TODO: use zip modtime when that is set correctly
				time: time.Now(),
			}
		}
		box.Files[fileName] = af

		// add to parent dir (if any)
		dirName := filepath.Dir(fileName)
		if dirName == "." {
			dirName = ""
		}
		if fileName != "" { // don't make box root dir a child of itself
			if dir := box.Files[dirName]; dir != nil {
				dir.children = append(dir.children, af)
			}
		}
	}
}

// implements os.FileInfo.
// used for Readdir()
type appendedDirInfo struct {
	name string
	time time.Time
}

func (adi *appendedDirInfo) Name() string {
	return adi.name
}
func (adi *appendedDirInfo) Size() int64 {
	return 0
}
func (adi *appendedDirInfo) Mode() os.FileMode {
	return os.ModeDir
}
func (adi *appendedDirInfo) ModTime() time.Time {
	return adi.time
}
func (adi *appendedDirInfo) IsDir() bool {
	return true
}
func (adi *appendedDirInfo) Sys() interface{} {
	return nil
}
