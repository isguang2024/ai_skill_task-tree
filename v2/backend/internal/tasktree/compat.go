package tasktree

import (
	"os"
	"path/filepath"
)

func osRemoveAll(path string) error {
	return os.RemoveAll(path)
}

func filepathJoin(elem ...string) string {
	return filepath.Join(elem...)
}

