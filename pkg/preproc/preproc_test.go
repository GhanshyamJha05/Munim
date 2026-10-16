package preproc

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareImage(t *testing.T) {
	tempDir := t.TempDir()
	samplePath := filepath.Join(tempDir, "sample_bill.jpg")

	// Create a dummy image 2000x1000 to test max dimension clamp
	img := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	for x := 0; x < 2000; x++ {
		for y := 0; y < 1000; y++ {
			img.Set(x, y, color.RGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}

	f, err := os.Create(samplePath)
	if err != nil {
		t.Fatalf("failed to create sample image: %v", err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("failed to encode sample image: %v", err)
	}
	f.Close()

	outDir := filepath.Join(tempDir, "proc")
	proc, err := PrepareImage(samplePath, outDir)
	if err != nil {
		t.Fatalf("PrepareImage failed: %v", err)
	}

	if proc.Width > MaxImageDimension {
		t.Errorf("Expected width <= %d, got %d", MaxImageDimension, proc.Width)
	}
	if proc.Height > MaxImageDimension {
		t.Errorf("Expected height <= %d, got %d", MaxImageDimension, proc.Height)
	}
	if len(proc.Base64Data) == 0 {
		t.Errorf("Expected non-empty base64 string")
	}
}
