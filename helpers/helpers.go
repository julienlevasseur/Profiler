package helpers

import (
	"os"
	"path/filepath"
)

/*ListFiles return a list of filenames that match the provided extension
found in the given folder*/
func ListFiles(folder string, extension string) []string {
	var files []string

	files, err := filepath.Glob(folder + "/" + extension)
	if err != nil {
		panic(err)
	}
	return files
}

/*CleanProfileFile remove the .profiler file*/
func CleanProfileFile() {
	err := os.Remove(".profiler")

	if err != nil {
		panic(err)
	}
}
