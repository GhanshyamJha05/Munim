package preproc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const MaxImageDimension = 1600

// ProcessedImage represents an image prepared for vision model consumption.
type ProcessedImage struct {
	OriginalPath   string
	SavedPath      string
	Base64Data     string // standard base64 string without data:image prefix
	Format         string // "jpeg" or "png"
	Width          int
	Height         int
	AspectRatio    float64
	Tiles          []string // Base64 overlapping vertical tiles if aspect ratio > 1.6
	PerceptualHash string   // 64-bit dHash representation (16 hex chars)
}

// PrepareImage loads, sanitizes, scales down if over MaxImageDimension, splits tall images into tiles, and returns base64.
func PrepareImage(filePath string, outputDir string) (*ProcessedImage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".pdf" {
		return nil, fmt.Errorf("direct PDF processing requires rendering to image; please upload as JPG/PNG or install pdftoppm")
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (%s): %w", format, err)
	}

	return processLoadedImage(img, filePath, outputDir)
}

// ProcessBytes processes raw multipart/form file upload bytes directly.
func ProcessBytes(r io.Reader, filename string, outputDir string) (*ProcessedImage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read bytes: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode uploaded image: %w", err)
	}

	return processLoadedImage(img, filename, outputDir)
}

func processLoadedImage(img image.Image, originalPath, outputDir string) (*ProcessedImage, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var targetImg image.Image = img
	if width > MaxImageDimension || height > MaxImageDimension {
		targetImg = resizeNearest(img, MaxImageDimension)
		bounds = targetImg.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, targetImg, &jpeg.Options{Quality: 88}); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	aspectRatio := float64(height) / float64(width)
	var tiles []string

	// Gemma Quirk #2: Fixed-resolution vision input downsamples tall bills to small squares.
	// If aspect ratio exceeds 1.6:1, slice into overlapping vertical tiles (~15% overlap)
	if aspectRatio > 1.6 {
		tiles = createVerticalTiles(targetImg, 2, 0.15)
	}

	var savedPath string
	if outputDir != "" {
		_ = os.MkdirAll(outputDir, 0755)
		cleanName := filepath.Base(originalPath)
		if filepath.Ext(cleanName) == "" {
			cleanName += ".jpg"
		}
		savedPath = filepath.Join(outputDir, cleanName)
		_ = os.WriteFile(savedPath, buf.Bytes(), 0644)
	}

	return &ProcessedImage{
		OriginalPath: originalPath,
		SavedPath:    savedPath,
		Base64Data:   base64.StdEncoding.EncodeToString(buf.Bytes()),
		Format:       "jpeg",
		Width:          width,
		Height:         height,
		AspectRatio:    aspectRatio,
		Tiles:          tiles,
		PerceptualHash: ComputeDHash(targetImg),
	}, nil
}

// ComputeDHash computes a 64-bit difference hash (dHash) for an image as a 16-hex character string.
func ComputeDHash(img image.Image) string {
	// Downscale to 9 columns x 8 rows grayscale
	gray := make([][]uint32, 8)
	for y := 0; y < 8; y++ {
		gray[y] = make([]uint32, 9)
		for x := 0; x < 9; x++ {
			// Sample pixel from source
			bounds := img.Bounds()
			srcX := bounds.Min.X + (x * bounds.Dx() / 9)
			srcY := bounds.Min.Y + (y * bounds.Dy() / 8)
			r, g, b, _ := img.At(srcX, srcY).RGBA()
			// Standard luminance: 0.299R + 0.587G + 0.114B
			lum := (r*299 + g*587 + b*114) / 1000
			gray[y][x] = lum
		}
	}

	var hash uint64
	bitIndex := 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if gray[y][x] > gray[y][x+1] {
				hash |= (1 << bitIndex)
			}
			bitIndex++
		}
	}
	return fmt.Sprintf("%016x", hash)
}

// HammingDistance calculates the bitwise Hamming distance between two 16-char hex hashes.
func HammingDistance(hash1, hash2 string) int {
	if len(hash1) != 16 || len(hash2) != 16 {
		return 64
	}
	var h1, h2 uint64
	fmt.Sscanf(hash1, "%016x", &h1)
	fmt.Sscanf(hash2, "%016x", &h2)
	diff := h1 ^ h2

	dist := 0
	for diff > 0 {
		dist += int(diff & 1)
		diff >>= 1
	}
	return dist
}

// createVerticalTiles slices targetImg into n vertical tiles with overlapPct overlap.
func createVerticalTiles(src image.Image, numTiles int, overlapPct float64) []string {
	bounds := src.Bounds()
	h := bounds.Dy()

	tileH := int(float64(h) / (float64(numTiles) - (float64(numTiles-1) * overlapPct)))
	step := int(float64(tileH) * (1.0 - overlapPct))

	var tiles []string
	for i := 0; i < numTiles; i++ {
		startY := bounds.Min.Y + i*step
		endY := startY + tileH
		if endY > bounds.Max.Y {
			endY = bounds.Max.Y
		}

		rect := image.Rect(bounds.Min.X, startY, bounds.Max.X, endY)
		tileImg := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				tileImg.Set(x-rect.Min.X, y-rect.Min.Y, src.At(x, y))
			}
		}

		var tileBuf bytes.Buffer
		_ = jpeg.Encode(&tileBuf, tileImg, &jpeg.Options{Quality: 88})
		tiles = append(tiles, base64.StdEncoding.EncodeToString(tileBuf.Bytes()))
	}
	return tiles
}

// Simple bilinear/nearest neighbor scaling for standard Go images without external C dependencies
func resizeNearest(src image.Image, maxDim int) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	var newW, newH int
	if w > h {
		newW = maxDim
		newH = int(float64(h) * float64(maxDim) / float64(w))
	} else {
		newH = maxDim
		newW = int(float64(w) * float64(maxDim) / float64(h))
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		srcY := bounds.Min.Y + (y * h / newH)
		for x := 0; x < newW; x++ {
			srcX := bounds.Min.X + (x * w / newW)
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

// Register image decoders
func init() {
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
}
