package tasktree

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
		addr = "127.0.0.1:8880"
	}
	server := &http.Server{
		Addr:    addr,
		Handler: app.HTTPHandler(),
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		log.Printf("shutdown signal received, stopping Task Tree service")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http shutdown failed: %v", err)
		}
		if err := app.Close(); err != nil {
			log.Printf("app close failed: %v", err)
		}
	}()
	log.Printf("Task Tree Go service listening on http://%s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	return 0
}
