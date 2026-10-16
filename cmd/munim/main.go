package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/munim-app/munim/pkg/db"
	"github.com/munim-app/munim/pkg/ollama"
	"github.com/munim-app/munim/pkg/pricealert"
	"github.com/munim-app/munim/pkg/server"
)

func main() {
	port := flag.Int("port", 8080, "Web server port")
	dbPath := flag.String("db", "data/munim.db", "SQLite database file path")
	uploadDir := flag.String("uploads", "data/bills", "Directory to store bill images")
	defaultModel := os.Getenv("GEMMA_MODEL")
	if defaultModel == "" {
		defaultModel = "gemma3:12b"
	}
	modelName := flag.String("model", defaultModel, "Ollama Gemma model tag (e.g. gemma3:12b, gemma3:4b, gemma3:27b)")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama daemon base URL")
	spikeThreshold := flag.Float64("threshold", 3.0, "Price spike alert threshold percentage")
	flag.Parse()

	log.Println("Starting Munim (मुनीम)...")

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize SQLite store
	store, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer store.Close()

	// Initialize Ollama client
	ollamaClient := ollama.NewClient(ollama.Config{
		BaseURL: *ollamaURL,
		Model:   *modelName,
	})

	// Pre-flight Ollama health check
	ctx := context.Background()
	log.Printf("Checking local Ollama connection (%s, model: %s)...", *ollamaURL, *modelName)
	if err := ollamaClient.CheckStatus(ctx); err != nil {
		fmt.Printf("\n[WARNING] Ollama check notification:\n%v\n\n(Munim will continue running web UI, but uploads require Ollama)\n\n", err)
	} else {
		log.Println("Ollama connection verified successfully.")
	}

	// Price alert engine
	alertEngine := pricealert.NewEngine(pricealert.Config{
		SpikeThresholdPercent: *spikeThreshold,
		MaxDaysCheaperElse:     45,
	})

	// Start embedded HTTP server
	srv := server.NewServer(server.Config{
		Port:        *port,
		Store:       store,
		Ollama:      ollamaClient,
		AlertEngine: alertEngine,
		UploadDir:   *uploadDir,
	})

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nShutting down Munim cleanly...")
		_ = store.Close()
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
