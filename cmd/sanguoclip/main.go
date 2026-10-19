package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/httpapi"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8787", "HTTP listen address")
	dataDir := flag.String("data-dir", "./data", "local data directory")
	flag.Parse()

	absoluteDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve data directory:", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	server, err := httpapi.New(absoluteDataDir, logger)
	if err != nil {
		logger.Error("initialize service", "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              *address,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logger.Info("SanguoClip is ready", "url", "http://"+*address, "data_dir", absoluteDataDir)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}
