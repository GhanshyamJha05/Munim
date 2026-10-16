package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/munim-app/munim/pkg/ollama"
	"github.com/munim-app/munim/pkg/preproc"
)

func main() {
	imagePath := flag.String("image", "", "Path to the bill image (JPG/PNG)")
	modelName := flag.String("model", "gemma3:12b", "Ollama model name (e.g. gemma3:12b or qwen2.5vl:7b)")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama daemon base URL")
	checkOnly := flag.Bool("check-ollama", false, "Verify Ollama connectivity and exit")
	flag.Parse()

	client := ollama.NewClient(ollama.Config{
		BaseURL: *ollamaURL,
		Model:   *modelName,
	})

	ctx := context.Background()

	if *checkOnly {
		log.Printf("Connecting to Ollama at %s (checking model %s)...", *ollamaURL, *modelName)
		if err := client.CheckStatus(ctx); err != nil {
			log.Fatalf("Ollama check failed:\n%v", err)
		}
		log.Println("SUCCESS: Ollama is running and model is available.")
		return
	}

	if *imagePath == "" {
		fmt.Println("Usage: munim-cli -image <path-to-bill> [-model gemma3:12b] [-ollama-url http://localhost:11434]")
		fmt.Println("       munim-cli -check-ollama")
		os.Exit(1)
	}

	log.Printf("Checking Ollama status...")
	if err := client.CheckStatus(ctx); err != nil {
		log.Fatalf("Ollama Pre-flight Check Error:\n%v", err)
	}

	log.Printf("Preprocessing bill image: %s", *imagePath)
	tempDir := filepath.Join(os.TempDir(), "munim_proc")
	procImg, err := preproc.PrepareImage(*imagePath, tempDir)
	if err != nil {
		log.Fatalf("Preprocessing failed: %v", err)
	}
	log.Printf("Image processed: %dx%d (%s, base64 length: %d)", procImg.Width, procImg.Height, procImg.Format, len(procImg.Base64Data))

	log.Printf("Extracting invoice data using %s via Ollama...", *modelName)
	extraction, err := client.ExtractBill(ctx, procImg.Base64Data, procImg.Tiles)
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	outJSON, _ := json.MarshalIndent(extraction, "", "  ")
	fmt.Println("\n=== EXTRACTED INVOICE ===")
	fmt.Println(string(outJSON))
}
