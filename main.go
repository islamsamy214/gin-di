package main

import (
	"log"
	"log/slog"
	"os"
	"web-app/app/providers"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// Start server on http
	if len(os.Args) > 1 && os.Args[1] == "http" {
		if err := providers.NewHTTPServiceProvider().Boot(); err != nil {
			/*
			 * slog rather than log.Fatalf. Boot installs the application's slog
			 * handler, and the stdlib log package bridges through it at INFO —
			 * so a failed boot was being recorded as an informational message,
			 * which any alert keyed on level=ERROR would miss entirely.
			 */
			slog.Error("http server failed to start", slog.Any("error", err))
			os.Exit(1)
		}

		/*
		 * The return is load-bearing. Without it an orderly shutdown fell through
		 * into the console dispatcher below, which found no command named "http"
		 * and exited 1 — so every clean stop looked like a crash to supervisord
		 * and to any orchestrator watching the exit code.
		 */
		return
	}

	// Any other commands should go as console commands
	if len(os.Args) >= 2 {
		providers.NewConsoleServiceProvider(os.Args[1], os.Args[2:]).Boot()
		os.Exit(0)
	}

	log.Println("Usage: go run main.go <command>")
	os.Exit(1)
}
