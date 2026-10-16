package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/munim-app/munim/pkg/db"
	"github.com/munim-app/munim/pkg/models"
	"github.com/munim-app/munim/pkg/pricealert"
	"github.com/munim-app/munim/pkg/validator"
)

type GroundTruthBill struct {
	BillID            int64                    `json:"bill_id"`
	FileName          string                   `json:"file_name"`
	Distributor       string                   `json:"distributor"`
	ExpectedData      models.InvoiceExtraction `json:"expected_data"`
	DeliberateAnomaly string                   `json:"deliberate_anomaly,omitempty"`
	ExpectedAlerts    []string                 `json:"expected_alerts,omitempty"`
}

func main() {
	dbPath := "data/munim.db"
	demoDir := "demo_data"
	truthPath := filepath.Join(demoDir, "ground_truth.json")

	log.Println("Seeding Munim database with realistic demo kirana bills from", truthPath)

	data, err := os.ReadFile(truthPath)
	if err != nil {
		log.Fatalf("Could not read ground truth file: %v. Please run 'make demo-data' first.", err)
	}

	var truths []GroundTruthBill
	if err := json.Unmarshal(data, &truths); err != nil {
		log.Fatalf("Failed to parse ground truth JSON: %v", err)
	}

	// Copy synthetic bill images to data/bills
	uploadDir := "data/bills"
	_ = os.RemoveAll(uploadDir)
	_ = os.MkdirAll(uploadDir, 0755)

	// Wipe existing SQLite database file so seeding is completely deterministic and idempotent
	_ = os.Remove(dbPath)

	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	alertEngine := pricealert.NewEngine(pricealert.DefaultConfig())

	// Insert bills in chronological order
	var seededCount int
	var seededAlerts int

	for _, gt := range truths {
		srcImg := filepath.Join(demoDir, "bills", gt.FileName)
		destImg := filepath.Join(uploadDir, gt.FileName)
		imgBytes, err := os.ReadFile(srcImg)
		if err == nil {
			_ = os.WriteFile(destImg, imgBytes, 0644)
		}

		invDate, _ := validator.ParseDate(gt.ExpectedData.InvoiceDate)
		var dueDate *time.Time
		if gt.ExpectedData.DueDate != nil && *gt.ExpectedData.DueDate != "" {
			d, err := validator.ParseDate(*gt.ExpectedData.DueDate)
			if err == nil {
				dueDate = &d
			}
		}

		valRes := validator.ValidateInvoice(&gt.ExpectedData)
		status := models.StatusVerified
		if !valRes.IsValid {
			status = models.StatusNeedsReview
		}

		// Some bills marked as paid, partial or unpaid to demonstrate dues
		paymentStatus := "unpaid"
		var amountPaid float64
		if gt.BillID <= 10 {
			paymentStatus = "paid"
			amountPaid = gt.ExpectedData.Total
		} else if gt.BillID <= 16 {
			paymentStatus = "partially_paid"
			amountPaid = mathRound(gt.ExpectedData.Total * 0.5)
		}

		bill := &models.BillRecord{
			VendorName:       gt.ExpectedData.VendorName,
			VendorGSTIN:      gt.ExpectedData.VendorGSTIN,
			InvoiceNumber:    gt.ExpectedData.InvoiceNumber,
			InvoiceDate:      invDate,
			DueDate:          dueDate,
			Subtotal:         gt.ExpectedData.Subtotal,
			CGST:             gt.ExpectedData.CGST,
			SGST:             gt.ExpectedData.SGST,
			IGST:             gt.ExpectedData.IGST,
			RoundOff:         gt.ExpectedData.RoundOff,
			Total:            gt.ExpectedData.Total,
			ImagePath:        gt.FileName,
			ValidationStatus: status,
			ValidationErrors: valRes.Errors,
			Confidence:       gt.ExpectedData.Confidence,
			LanguageDetected: gt.ExpectedData.LanguageDetected,
			LineItems:        gt.ExpectedData.LineItems,
			AmountPaid:       amountPaid,
			PaymentStatus:    paymentStatus,
		}

		billID, err := store.InsertBill(ctx, bill)
		if err != nil {
			log.Fatalf("Failed to insert bill %d: %v", gt.BillID, err)
		}
		seededCount++

		// Check price alerts across history up to this bill
		history, _ := store.GetHistoricalPurchases(ctx)
		for _, item := range gt.ExpectedData.LineItems {
			alerts := alertEngine.CheckItemPrice(billID, gt.ExpectedData.InvoiceNumber, gt.FileName, invDate, gt.ExpectedData.VendorName, item, history)
			for _, a := range alerts {
				_ = store.SavePriceAlert(ctx, a)
				seededAlerts++
			}
		}
	}

	fmt.Printf("\nSUCCESS: Seeded %d bills and %d active price alerts into %s\n", seededCount, seededAlerts, dbPath)
}

func mathRound(val float64) float64 {
	return float64(int(val*100)) / 100
}
