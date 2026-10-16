package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/munim-app/munim/pkg/models"
)

func TestStoreOperations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_munim.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 1. Insert bill
	dueDate := time.Now().AddDate(0, 0, 15)
	bill := &models.BillRecord{
		VendorName:       "Sharma Distributors",
		VendorGSTIN:      "07AAAAA0000A1Z5",
		InvoiceNumber:    "SD-2026-101",
		InvoiceDate:      time.Now(),
		DueDate:          &dueDate,
		Subtotal:         1000.0,
		CGST:             90.0,
		SGST:             90.0,
		Total:            1180.0,
		ImagePath:        "test.jpg",
		ValidationStatus: models.StatusVerified,
		Confidence:       0.98,
		LineItems: []models.LineItem{
			{Description: "Parle-G 12-pack", UnitPrice: 118.0, Quantity: 10, TaxableValue: 1000.0},
		},
		PaymentStatus: "unpaid",
	}

	id, err := store.InsertBill(ctx, bill)
	if err != nil {
		t.Fatalf("InsertBill failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	// 2. Fetch all bills
	bills, err := store.GetAllBills(ctx)
	if err != nil {
		t.Fatalf("GetAllBills failed: %v", err)
	}
	if len(bills) != 1 {
		t.Fatalf("Expected 1 bill, got %d", len(bills))
	}

	// 3. Record payment
	err = store.RecordPayment(ctx, id, 500.0)
	if err != nil {
		t.Fatalf("RecordPayment failed: %v", err)
	}

	// 4. Verify dues
	dues, err := store.GetSupplierDues(ctx)
	if err != nil {
		t.Fatalf("GetSupplierDues failed: %v", err)
	}
	if len(dues) != 1 {
		t.Fatalf("Expected 1 supplier due summary, got %d", len(dues))
	}
	if dues[0].BalanceDue != 680.0 { // 1180 - 500
		t.Errorf("Expected balance due 680.0, got %.2f", dues[0].BalanceDue)
	}
}
