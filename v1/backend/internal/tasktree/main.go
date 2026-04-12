package tasktree

import (
	"fmt"
	"log"
	"os"
)

func Run(args []string) int {
	mode := "serve"
	if len(args) > 0 {
		mode = args[0]
	}
	if mode == "client" {
		return runClient(args[1:])
	}
	if mode == "healthcheck" {
		return runHealthcheck()
	}
	if mode == "mcp" {
		app, err := NewApp()
		if err != nil {
			log.Fatal(err)
		}
		if err := runMCP(app, os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return 0
	}
	if mode != "serve" {
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", mode)
		return 1
	}
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("TTS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8879"
	}
	log.Printf("Task Tree Go service listening on http://%s", addr)
	log.Fatal(app.ListenAndServe(addr))
	return 0
}

