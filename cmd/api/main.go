package main

import (
	"log/slog"
	"os"
)

func main() {
	println("Hello World!")

	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("BaaS backend is starting...")
}
