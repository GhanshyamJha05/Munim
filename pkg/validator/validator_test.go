package validator

import (
	"strings"
	"testing"

	"github.com/munim-app/munim/pkg/models"
)

// Helper to compute correct GSTIN checksum for testing
func computeGSTINChecksum(first14 string) string {
	first14 = strings.ToUpper(first14)
	var sum int
	for i := 0; i < 14; i++ {
		ch := first14[i]
		idx := strings.IndexByte(gstinChars, ch)
		factor := 1
		if i%2 == 1 {
			factor = 2
		}
		digit := idx * factor
		digit = (digit / 36) + (digit % 36)
		sum += digit
	}
	checkCode := (36 - (sum % 36)) % 36
	return string(gstinChars[checkCode])
}

func TestGSTINValidator(t *testing.T) {
	// Let's generate a couple of valid GSTINs
	base1 := "07AAAAA0000A1Z"
	c1 := computeGSTINChecksum(base1)
	validGSTIN1 := base1 + c1

	base2 := "27ABCDE1234F2Z"
	c2 := computeGSTINChecksum(base2)
	validGSTIN2 := base2 + c2

	tests := []struct {
		name      string
		gstin     string
		wantValid bool
	}{
		{"Valid GSTIN 1", validGSTIN1, true},
		{"Valid GSTIN 2", validGSTIN2, true},
		{"Invalid Checksum", base1 + "9", false},
		{"Short Length", "07AAAAA0000A1Z", false},
		{"Illegal Characters", "07AAAAA0000A1Z@", false},
		{"Empty GSTIN", "", false},
		{"Lowercase handled", strings.ToLower(validGSTIN1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errStr := ValidateGSTIN(tt.gstin)
			if got != tt.wantValid {
				t.Errorf("ValidateGSTIN(%q) = %v, want %v (error: %s)", tt.gstin, got, tt.wantValid, errStr)
			}
		})
	}
}

func TestValidateInvoice(t *testing.T) {
	baseGSTIN := "07AAAAA0000A1Z"
	validGSTIN := baseGSTIN + computeGSTINChecksum(baseGSTIN)

	tests := []struct {
		name        string
		inv         models.InvoiceExtraction
		wantValid   bool
		expectError string
	}{
		{
			name: "Perfect Valid Invoice",
			inv: models.InvoiceExtraction{
				VendorName:       "Sharma Distributors",
				VendorGSTIN:      validGSTIN,
				InvoiceNumber:    "INV-2026-001",
				InvoiceDate:      "2026-01-15",
				Subtotal:         1000.00,
				CGST:             90.00,
				SGST:             90.00,
				IGST:             0.0,
				RoundOff:         0.0,
				Total:            1180.00,
				Confidence:       0.95,
				LanguageDetected: "English",
				LineItems: []models.LineItem{
					{Description: "Parle-G 12-pack", TaxableValue: 500.00},
					{Description: "Tata Salt 1kg", TaxableValue: 500.00},
				},
			},
			wantValid: true,
		},
		{
			name: "Line items sum mismatch subtotal",
			inv: models.InvoiceExtraction{
				VendorName:    "Sharma Distributors",
				VendorGSTIN:   validGSTIN,
				InvoiceNumber: "INV-2026-002",
				InvoiceDate:   "2026-01-15",
				Subtotal:      1000.00,
				CGST:          90.00,
				SGST:          90.00,
				Total:         1180.00,
				Confidence:    0.95,
				LineItems: []models.LineItem{
					{Description: "Parle-G 12-pack", TaxableValue: 400.00}, // sum 400 vs subtotal 1000
				},
			},
			wantValid:   false,
			expectError: "Line items taxable total",
		},
		{
			name: "Taxes plus Subtotal mismatch Grand Total",
			inv: models.InvoiceExtraction{
				VendorName:    "Sharma Distributors",
				VendorGSTIN:   validGSTIN,
				InvoiceNumber: "INV-2026-003",
				InvoiceDate:   "2026-01-15",
				Subtotal:      1000.00,
				CGST:          90.00,
				SGST:          90.00,
				Total:         1500.00, // Arithmetic error
				Confidence:    0.95,
				LineItems: []models.LineItem{
					{Description: "Parle-G 12-pack", TaxableValue: 1000.00},
				},
			},
			wantValid:   false,
			expectError: "does not match Grand Total",
		},
		{
			name: "Intrastate CGST not equal SGST",
			inv: models.InvoiceExtraction{
				VendorName:    "Sharma Distributors",
				VendorGSTIN:   validGSTIN,
				InvoiceNumber: "INV-2026-004",
				InvoiceDate:   "2026-01-15",
				Subtotal:      1000.00,
				CGST:          90.00,
				SGST:          50.00, // mismatch
				IGST:          0,
				Total:         1140.00,
				Confidence:    0.95,
				LineItems: []models.LineItem{
					{Description: "Parle-G 12-pack", TaxableValue: 1000.00},
				},
			},
			wantValid:   false,
			expectError: "Intra-state tax mismatch",
		},
		{
			name: "Confidence below 0.70 threshold",
			inv: models.InvoiceExtraction{
				VendorName:    "Sharma Distributors",
				VendorGSTIN:   validGSTIN,
				InvoiceNumber: "INV-2026-005",
				InvoiceDate:   "2026-01-15",
				Subtotal:      1000.00,
				CGST:          90.00,
				SGST:          90.00,
				Total:         1180.00,
				Confidence:    0.55, // Low confidence
				LineItems: []models.LineItem{
					{Description: "Parle-G 12-pack", TaxableValue: 1000.00},
				},
			},
			wantValid:   false,
			expectError: "Extraction confidence 0.55 is below threshold 0.70",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ValidateInvoice(&tt.inv)
			if res.IsValid != tt.wantValid {
				t.Errorf("ValidateInvoice() isValid = %v, want %v (errors: %v)", res.IsValid, tt.wantValid, res.Errors)
			}
			if tt.expectError != "" {
				joined := strings.Join(res.Errors, "; ")
				if !strings.Contains(joined, tt.expectError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectError, joined)
				}
			}
		})
	}
}
