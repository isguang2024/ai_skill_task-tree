package main

import (
	"os"

	"task-tree-service-v2/internal/tasktree"
)

func main() {
	os.Exit(tasktree.Run(os.Args[1:]))
}
