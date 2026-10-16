package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/munim-app/munim/pkg/db"
	"github.com/munim-app/munim/pkg/models"
)

// TestMonthlySpendDifferentMonths asserts that bills in different months are properly separated.
func TestMonthlySpendDifferentMonths(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_spend.db")

	store, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	janDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	febDate := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	// Jan Bill
	_, err = store.InsertBill(ctx, &models.BillRecord{
		VendorName:       "Sharma Distributors",
		VendorGSTIN:      "07AAAAA0000A1Z5",
		InvoiceNumber:    "JAN-001",
		InvoiceDate:      janDate,
		Subtotal:         5000.0,
		Total:            5900.0,
		ValidationStatus: models.StatusVerified,
		PaymentStatus:    "unpaid",
	})
	if err != nil {
		t.Fatalf("Failed to insert Jan bill: %v", err)
	}

	// Feb Bill
	_, err = store.InsertBill(ctx, &models.BillRecord{
		VendorName:       "Bharat Trading Co",
		VendorGSTIN:      "07BBBCC1111B1ZF",
		InvoiceNumber:    "FEB-001",
		InvoiceDate:      febDate,
		Subtotal:         10000.0,
		Total:            11800.0,
		ValidationStatus: models.StatusVerified,
		PaymentStatus:    "unpaid",
	})
	if err != nil {
		t.Fatalf("Failed to insert Feb bill: %v", err)
	}

	bills, err := store.GetAllBills(ctx)
	if err != nil {
		t.Fatalf("Failed to get bills: %v", err)
	}
	if len(bills) != 2 {
		t.Fatalf("Expected 2 bills, got %d", len(bills))
	}

	// Calculate spend for Jan 2026
	var janSpend float64
	var febSpend float64
	for _, b := range bills {
		if b.ValidationStatus != "rejected" {
			monthStr := b.InvoiceDate.Format("2006-01")
			if monthStr == "2026-01" {
				janSpend += b.Total
			} else if monthStr == "2026-02" {
				febSpend += b.Total
			}
		}
	}

	if janSpend != 5900.0 {
		t.Errorf("Expected Jan spend 5900.00, got %.2f", janSpend)
	}
	if febSpend != 11800.0 {
		t.Errorf("Expected Feb spend 11800.00, got %.2f", febSpend)
	}
}

// TestIdempotentSeedingBillCountAndMonthlySpend asserts bill count equals generator's count and spend equals monthly sum
func TestIdempotentSeedingBillCountAndMonthlySpend(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_idempotent.db")

	store, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Seed 30 simulated bills across 2 months (e.g. 2026-03 and 2026-04)
	baseDate := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	var expectedMarchTotal float64
	var marchCount int
	totalBillsToGen := 30

	for i := 0; i < totalBillsToGen; i++ {
		invDate := baseDate.AddDate(0, 0, i*2)
		billTotal := float64(1000 + (i * 100))
		if invDate.Month() == time.March {
			expectedMarchTotal += billTotal
			marchCount++
		}

		bill := &models.BillRecord{
			VendorName:       "Sharma Distributors",
			VendorGSTIN:      "07AAAAA0000A1Z5",
			InvoiceNumber:    time.Now().Format("20060102150405") + string(rune('A'+i)),
			InvoiceDate:      invDate,
			Subtotal:         billTotal * 0.82,
			Total:            billTotal,
			ValidationStatus: models.StatusVerified,
			PaymentStatus:    "unpaid",
		}
		_, err := store.InsertBill(ctx, bill)
		if err != nil {
			t.Fatalf("Failed to insert bill %d: %v", i+1, err)
		}
	}

	bills, err := store.GetAllBills(ctx)
	if err != nil {
		t.Fatalf("Failed to get bills: %v", err)
	}

	// 1. Assert bill count equals generator's count
	if len(bills) != totalBillsToGen {
		t.Errorf("Expected exactly %d bills, got %d", totalBillsToGen, len(bills))
	}

	// 2. Assert spend for March equals exact sum of March bills
	var actualMarchSpend float64
	for _, b := range bills {
		if b.ValidationStatus != "rejected" && b.InvoiceDate.Format("2006-01") == "2026-03" {
			actualMarchSpend += b.Total
		}
	}

	if actualMarchSpend != expectedMarchTotal {
		t.Errorf("March spend mismatch: expected %.2f, got %.2f", expectedMarchTotal, actualMarchSpend)
	}

	// 3. Assert UNIQUE constraint prevents duplicates
	duplicateBill := &models.BillRecord{
		VendorName:       bills[0].VendorName,
		VendorGSTIN:      bills[0].VendorGSTIN,
		InvoiceNumber:    bills[0].InvoiceNumber,
		InvoiceDate:      bills[0].InvoiceDate,
		Subtotal:         bills[0].Subtotal,
		Total:            bills[0].Total,
		ValidationStatus: models.StatusVerified,
		PaymentStatus:    "unpaid",
	}
	_, dupErr := store.InsertBill(ctx, duplicateBill)
	if dupErr == nil {
		t.Errorf("Expected UNIQUE constraint violation on duplicate bill, but insert succeeded")
	}
}
