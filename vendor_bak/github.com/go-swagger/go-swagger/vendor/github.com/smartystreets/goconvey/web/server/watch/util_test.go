// Credits: https://gist.github.com/jaybill/2876519
package watch

import "os"
import "io"
import "io/ioutil"
import "log"

// Copies original source to destination destination.
func CopyFile(source string, destination string) (err error) {
	originalFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer originalFile.Close()
	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()
	_, err = io.Copy(destinationFile, originalFile)
	if err == nil {
		info, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(destination, info.Mode())
		}

	}

	return
}

// Recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
func CopyDir(source string, destination string) (err error) {

	// get properties of source dir
	sourceFile, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !sourceFile.IsDir() {
		return &CustomError{Message: "Source is not a directory"}
	}

	// ensure destination dir does not already exist

	_, err = os.Open(destination)
	if !os.IsNotExist(err) {
		return &CustomError{Message: "Destination already exists"}
	}

	// create destination dir

	err = os.MkdirAll(destination, sourceFile.Mode())
	if err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(source)

	for _, entry := range entries {

		sourcePath := source + "/" + entry.Name()
		destinationPath := destination + "/" + entry.Name()
		if entry.IsDir() {
			err = CopyDir(sourcePath, destinationPath)
			if err != nil {
				log.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sourcePath, destinationPath)
			if err != nil {
				log.Println(err)
			}
		}

	}
	return
}

// A struct for returning custom error messages
type CustomError struct {
	Message string
}

// Returns the error message defined in Message as a string
func (this *CustomError) Error() string {
	return this.Message
}
