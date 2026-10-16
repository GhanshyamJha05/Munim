package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/munim-app/munim/pkg/matcher"
	"github.com/munim-app/munim/pkg/models"
)

// Supported Gemma tags
const (
	ModelGemma3_4B  = "gemma3:4b"
	ModelGemma3_12B = "gemma3:12b" // Default
	ModelGemma3_27B = "gemma3:27b"
)

// Config defines connection details for the local Ollama instance running Gemma.
type Config struct {
	BaseURL string // default: http://localhost:11434
	Model   string // default: gemma3:12b
	Timeout time.Duration
}

// Client wraps HTTP communication with local Ollama daemon for Gemma.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient returns an Ollama client configured with Gemma defaults.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Model == "" {
		// Allow override via GEMMA_MODEL env var, fallback to gemma3:12b
		if envModel := os.Getenv("GEMMA_MODEL"); envModel != "" {
			cfg.Model = envModel
		} else {
			cfg.Model = ModelGemma3_12B
		}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 180 * time.Second
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// ModelName returns the active Gemma model identifier.
func (c *Client) ModelName() string {
	return c.cfg.Model
}

// SetModel updates active model tag at runtime.
func (c *Client) SetModel(model string) {
	c.cfg.Model = model
}

// CheckStatus verifies if Ollama is running and whether the specified Gemma tag is pulled.
func (c *Client) CheckStatus(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create status request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s.\nPlease start Ollama with:\n    ollama serve\nError: %w", c.cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned unexpected HTTP status: %d", resp.StatusCode)
	}

	var tagsResp struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return fmt.Errorf("failed to parse ollama model list: %w", err)
	}

	found := false
	targetModel := strings.ToLower(c.cfg.Model)
	var available []string
	for _, m := range tagsResp.Models {
		available = append(available, m.Name)
		if strings.ToLower(m.Name) == targetModel || strings.HasPrefix(strings.ToLower(m.Name), targetModel+":") || strings.HasPrefix(targetModel, strings.ToLower(m.Name)+":") {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Gemma model %q is not pulled in Ollama.\nAvailable models: %v\nPlease pull it using:\n    ollama pull %s\nAlternative Gemma models:\n    ollama pull gemma3:4b  (low-RAM laptops)\n    ollama pull gemma3:27b (high-accuracy mode)",
			c.cfg.Model, available, c.cfg.Model)
	}

	return nil
}

// ExtractionJSONSchema defines the JSON schema passed to Ollama format parameter.
var ExtractionJSONSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"vendor_name": map[string]interface{}{"type": []string{"string", "null"}},
		"vendor_gstin": map[string]interface{}{"type": []string{"string", "null"}},
		"invoice_number": map[string]interface{}{"type": []string{"string", "null"}},
		"invoice_date": map[string]interface{}{"type": []string{"string", "null"}, "description": "ISO 8601 YYYY-MM-DD"},
		"due_date": map[string]interface{}{"type": []string{"string", "null"}, "description": "ISO 8601 or null"},
		"line_items": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{"type": "string"},
					"description_normalized": map[string]interface{}{"type": "string"},
					"hsn_code": map[string]interface{}{"type": "string"},
					"pack_size": map[string]interface{}{"type": "string"},
					"unit": map[string]interface{}{"type": "string"},
					"quantity": map[string]interface{}{"type": "number"},
					"unit_price": map[string]interface{}{"type": "number"},
					"taxable_value": map[string]interface{}{"type": "number"},
					"gst_rate": map[string]interface{}{"type": "number"},
				},
				"required": []string{"description", "quantity", "unit_price", "taxable_value"},
			},
		},
		"subtotal": map[string]interface{}{"type": "number"},
		"cgst": map[string]interface{}{"type": "number"},
		"sgst": map[string]interface{}{"type": "number"},
		"igst": map[string]interface{}{"type": "number"},
		"round_off": map[string]interface{}{"type": "number"},
		"total": map[string]interface{}{"type": "number"},
		"language_detected": map[string]interface{}{"type": "string"},
		"confidence": map[string]interface{}{"type": "number", "description": "0.0 to 1.0 confidence score"},
	},
	"required": []string{"vendor_name", "vendor_gstin", "invoice_number", "invoice_date", "line_items", "subtotal", "total"},
}

// LineItemsOnlyJSONSchema for tile extraction
var LineItemsOnlyJSONSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"line_items": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{"type": "string"},
					"description_normalized": map[string]interface{}{"type": "string"},
					"hsn_code": map[string]interface{}{"type": "string"},
					"pack_size": map[string]interface{}{"type": "string"},
					"unit": map[string]interface{}{"type": "string"},
					"quantity": map[string]interface{}{"type": "number"},
					"unit_price": map[string]interface{}{"type": "number"},
					"taxable_value": map[string]interface{}{"type": "number"},
					"gst_rate": map[string]interface{}{"type": "number"},
				},
				"required": []string{"description", "quantity", "unit_price", "taxable_value"},
			},
		},
	},
	"required": []string{"line_items"},
}

// LoadPrompt loads prompt text from file with fallback default.
func LoadPrompt(filePath string, fallback string) string {
	b, err := os.ReadFile(filePath)
	if err == nil && len(strings.TrimSpace(string(b))) > 0 {
		return string(b)
	}
	return fallback
}

var defaultExtractPrompt = `You are reading a photo of an Indian supplier bill or GST invoice. The text may be in English, Hindi (Devanagari), or a mix, and may be handwritten, faded, rotated or stamped.

Extract the data into the JSON schema provided. Rules:
1. Return null for any field you cannot read clearly. Never guess or invent values, especially GSTIN, invoice number and amounts.
2. Copy numbers exactly as printed. Indian digit grouping (1,23,456.00) is normal. Convert Devanagari digits to ASCII digits.
3. Dates are day-first (DD/MM/YYYY) unless clearly stated otherwise. Output ISO 8601 (YYYY-MM-DD).
4. One line_items entry per product row. Do not merge rows, do not add rows that are not on the bill. Keep description exactly as written, then give description_normalized in lowercase English with pack size and unit separated (e.g. "parle-g biscuit", pack_size 800, unit "g").
5. Do not calculate totals yourself. Copy subtotal, taxes, round_off and total as printed. If a total is handwritten, read it and lower its confidence.
6. For every top-level field give a confidence from 0 to 1 reflecting how legible it was.
7. Output only the JSON object, nothing else.`

var defaultRepairPrompt = `Your previous extraction failed this check: {validation_error}.
Look at the image again, focusing only on: {affected_fields}.
Return the full corrected JSON. If the value is still unreadable, return null for it rather than adjusting it to make the math work.`

// ExtractBill handles Gemma extraction respecting Gemma 3 quirks:
// - No system turn (instructions in user turn).
// - Deterministic decoding (temperature: 0, fixed seed: 42, num_ctx: 8192).
// - Multi-tile extraction when bill is tall (aspect ratio > 1.6:1).
func (c *Client) ExtractBill(ctx context.Context, fullBase64 string, tiles []string) (*models.InvoiceExtraction, error) {
	promptText := LoadPrompt("prompts/extract_v1.txt", defaultExtractPrompt)

	// Step 1: Extract header, totals and overall invoice from full image
	start := time.Now()
	extraction, err := c.callGemma(ctx, fullBase64, promptText, ExtractionJSONSchema)
	elapsed := time.Since(start)
	log.Printf("[Gemma %s] Full image extraction latency: %.2fs", c.cfg.Model, elapsed.Seconds())
	if err != nil {
		return nil, err
	}

	// Step 2: If bill is tall and vertical tiles exist, extract line items per tile and merge
	if len(tiles) > 0 {
		var mergedItems []models.LineItem
		tilePrompt := `Extract only the product line items visible in this section of the invoice into the line_items schema. Rules:
1. One line_items entry per product row.
2. Copy numbers and descriptions exactly as printed.
3. Output only the JSON object with key "line_items".`

		for tileIdx, tileB64 := range tiles {
			tileStart := time.Now()
			tileExt, tileErr := c.callGemma(ctx, tileB64, tilePrompt, LineItemsOnlyJSONSchema)
			log.Printf("[Gemma %s] Tile %d/%d extraction latency: %.2fs", c.cfg.Model, tileIdx+1, len(tiles), time.Since(tileStart).Seconds())
			if tileErr == nil && len(tileExt.LineItems) > 0 {
				for _, newItem := range tileExt.LineItems {
					// De-duplicate items from overlap zone
					if !isItemDuplicate(newItem, mergedItems) {
						mergedItems = append(mergedItems, newItem)
					}
				}
			}
		}

		if len(mergedItems) > len(extraction.LineItems) {
			extraction.LineItems = mergedItems
		}
	}

	return extraction, nil
}

// RepairBill handles Gemma 1-shot repair.
func (c *Client) RepairBill(ctx context.Context, base64Image string, priorJSON string, validationErrors []string) (*models.InvoiceExtraction, error) {
	repairTmpl := LoadPrompt("prompts/repair_v1.txt", defaultRepairPrompt)

	affected := "subtotal, line_items, total, cgst, sgst, igst, vendor_gstin"
	errMsg := strings.Join(validationErrors, "; ")

	userPrompt := strings.ReplaceAll(repairTmpl, "{validation_error}", errMsg)
	userPrompt = strings.ReplaceAll(userPrompt, "{affected_fields}", affected)
	userPrompt += fmt.Sprintf("\n\nPrior extracted JSON:\n%s", priorJSON)

	start := time.Now()
	res, err := c.callGemma(ctx, base64Image, userPrompt, ExtractionJSONSchema)
	log.Printf("[Gemma %s] Repair prompt latency: %.2fs", c.cfg.Model, time.Since(start).Seconds())
	return res, err
}

func (c *Client) callGemma(ctx context.Context, base64Image string, userText string, formatSchema interface{}) (*models.InvoiceExtraction, error) {
	url := c.cfg.BaseURL + "/api/chat"

	type Message struct {
		Role    string   `json:"role"`
		Content string   `json:"content"`
		Images  []string `json:"images,omitempty"`
	}

	type ChatRequest struct {
		Model    string                 `json:"model"`
		Messages []Message              `json:"messages"`
		Stream   bool                   `json:"stream"`
		Format   interface{}            `json:"format"`
		Options  map[string]interface{} `json:"options,omitempty"`
	}

	// Gemma Quirk #1: No system role! All instructions in the user turn with the image.
	// Gemma Quirk #3: Deterministic decoding: temperature 0, seed 42, num_ctx 8192 explicitly.
	reqBody := ChatRequest{
		Model: c.cfg.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: userText,
				Images:  []string{base64Image},
			},
		},
		Stream: false,
		Format: formatSchema,
		Options: map[string]interface{}{
			"temperature": 0.0,
			"seed":        42,
			"num_ctx":     8192,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var extraction models.InvoiceExtraction
	if err := json.Unmarshal([]byte(chatResp.Message.Content), &extraction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal extraction JSON: %w (content: %s)", err, chatResp.Message.Content)
	}

	return &extraction, nil
}

func isItemDuplicate(candidate models.LineItem, existing []models.LineItem) bool {
	normCand := matcher.NormalizeItem(candidate.Description, candidate.PackSize, candidate.HSNCode)
	for _, ex := range existing {
		normEx := matcher.NormalizeItem(ex.Description, ex.PackSize, ex.HSNCode)
		if matched, _ := matcher.AreItemsMatch(normCand, normEx); matched {
			if candidate.UnitPrice == ex.UnitPrice && candidate.Quantity == ex.Quantity {
				return true
			}
		}
	}
	return false
}
