package export

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/munim-app/munim/pkg/models"
)

func TestCSVExporters(t *testing.T) {
	bills := []models.BillRecord{
		{
			ID:               1,
			VendorName:       "Sharma Distributors",
			VendorGSTIN:      "07AAAAA0000A1Z5",
			InvoiceNumber:    "SD-2026-901",
			InvoiceDate:      time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
			Subtotal:         1000.0,
			CGST:             90.0,
			SGST:             90.0,
			Total:            1180.0,
			ValidationStatus: models.StatusVerified,
			PaymentStatus:    "unpaid",
			LineItems: []models.LineItem{
				{
					Description:  "Parle-G 12-pack",
					HSNCode:      "1905",
					PackSize:     "12pack",
					Quantity:     10,
					Unit:         "pack",
					UnitPrice:    100.0,
					TaxableValue: 1000.0,
					GSTRate:      18.0,
				},
			},
		},
	}

	t.Run("Test CA Pack CSV generation", func(t *testing.T) {
		data, err := GenerateCAPack(bills, time.Time{})
		if err != nil {
			t.Fatalf("GenerateCAPack failed: %v", err)
		}

		r := csv.NewReader(bytes.NewReader(data))
		records, err := r.ReadAll()
		if err != nil {
			t.Fatalf("Failed to read generated CSV: %v", err)
		}

		if len(records) != 2 { // Header + 1 row
			t.Fatalf("Expected 2 rows in CA Pack, got %d", len(records))
		}

		if records[1][0] != "SD-2026-901" || records[1][2] != "Sharma Distributors" {
			t.Errorf("Unexpected record content: %v", records[1])
		}
	})

	t.Run("Test Tally Voucher CSV generation", func(t *testing.T) {
		data, err := GenerateTallyVouchers(bills, time.Time{})
		if err != nil {
			t.Fatalf("GenerateTallyVouchers failed: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "Purchase") || !strings.Contains(content, "Sharma Distributors") {
			t.Errorf("Tally CSV missing essential voucher markers: %s", content)
		}
	})
}
