package tasktree

import (
	"os"
	"path/filepath"
)

func resolveUpwardPath(rel string) string {
	dir, err := os.Getwd()
	if err != nil {
		return rel
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Clean(filepath.Join(dir, rel))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return filepath.Clean(filepath.Join(dir, rel))
}
