package main

import (
	"os"

	"task-tree-service/internal/tasktree"
)

func main() {
	os.Exit(tasktree.Run(os.Args[1:]))
}
