package pricealert

import (
	"testing"
	"time"

	"github.com/munim-app/munim/pkg/models"
)

func TestPriceAlertEngine(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	bill1Date := now.AddDate(0, 0, -30)
	bill2Date := now.AddDate(0, 0, -15)
	bill3Date := now.AddDate(0, 0, -5)

	// Historical bills from Sharma Distributors
	history := []HistoricalPurchase{
		{
			BillID:        101,
			InvoiceNumber: "INV-2026-101",
			ImagePath:     "bill_101.jpg",
			BillDate:      bill1Date,
			SupplierName:  "Sharma Distributors",
			ItemDesc:      "Parle-G 12-pack",
			PackSize:      "12-pack",
			UnitPrice:     118.00,
			Quantity:      10,
		},
		{
			BillID:        102,
			InvoiceNumber: "INV-2026-102",
			ImagePath:     "bill_102.jpg",
			BillDate:      bill2Date,
			SupplierName:  "Sharma Distributors",
			ItemDesc:      "Parle-G 12-pack",
			PackSize:      "12-pack",
			UnitPrice:     118.00,
			Quantity:      10,
		},
		{
			BillID:        103,
			InvoiceNumber: "INV-2026-103",
			ImagePath:     "bill_103.jpg",
			BillDate:      bill3Date,
			SupplierName:  "Sharma Distributors",
			ItemDesc:      "Parle-G 12-pack",
			PackSize:      "12-pack",
			UnitPrice:     118.00,
			Quantity:      10,
		},
		// Historical for Fortune Sunflower Oil (Kailash)
		{
			BillID:        110,
			InvoiceNumber: "INV-2026-110",
			ImagePath:     "bill_110.jpg",
			BillDate:      bill2Date,
			SupplierName:  "Kailash Agro Foods",
			ItemDesc:      "Fortune Sunlite Sunflower Oil 1L",
			PackSize:      "1l",
			UnitPrice:     135.00,
			Quantity:      15,
		},
		// Historical for Atta (Bharat)
		{
			BillID:        120,
			InvoiceNumber: "INV-2026-120",
			ImagePath:     "bill_120.jpg",
			BillDate:      bill2Date,
			SupplierName:  "Bharat Trading Co",
			ItemDesc:      "Aashirvaad Shudh Chakki Atta 10kg",
			PackSize:      "10kg",
			UnitPrice:     380.00,
			Quantity:      10,
		},
		// Historical for Surf Excel (Sharma)
		{
			BillID:        130,
			InvoiceNumber: "INV-2026-130",
			ImagePath:     "bill_130.jpg",
			BillDate:      bill2Date,
			SupplierName:  "Sharma Distributors",
			ItemDesc:      "Surf Excel Quick Wash 1kg",
			PackSize:      "1kg",
			UnitPrice:     145.00,
			Quantity:      10,
		},
		// Historical for Colgate (Bharat)
		{
			BillID:        140,
			InvoiceNumber: "INV-2026-140",
			ImagePath:     "bill_140.jpg",
			BillDate:      bill2Date,
			SupplierName:  "Bharat Trading Co",
			ItemDesc:      "Colgate Strong Teeth 200g",
			PackSize:      "200g",
			UnitPrice:     110.00,
			Quantity:      20,
		},
		// Another supplier selling Tata Salt cheaper
		{
			BillID:        201,
			InvoiceNumber: "INV-2026-201",
			ImagePath:     "bill_201.jpg",
			BillDate:      bill2Date,
			SupplierName:  "Bharat Trading Co",
			ItemDesc:      "Tata Salt 1kg",
			PackSize:      "1kg",
			UnitPrice:     24.00,
			Quantity:      25,
		},
	}

	t.Run("Hike 1 (Parle-G): 118 -> 123 (+4.2%) with dual evidence and invoice numbers", func(t *testing.T) {
		newItem := models.LineItem{
			Description:  "Parle-G 12-pack Biscuit",
			PackSize:     "12-pack",
			UnitPrice:    123.00,
			Quantity:     10,
			TaxableValue: 1230.00,
		}

		alerts := engine.CheckItemPrice(104, "INV-2026-104", "bill_104.jpg", now, "Sharma Distributors", newItem, history)
		if len(alerts) != 1 {
			t.Fatalf("Expected exactly 1 consolidated alert, got %d", len(alerts))
		}

		alert := alerts[0]
		if alert.Type != "spike_both" {
			t.Errorf("Expected spike_both type when higher than both prior and median, got %s", alert.Type)
		}
		if alert.CurrentInvoiceNumber != "INV-2026-104" {
			t.Errorf("Expected CurrentInvoiceNumber INV-2026-104, got %s", alert.CurrentInvoiceNumber)
		}
		if alert.ReferenceInvoiceNumber != "INV-2026-103" {
			t.Errorf("Expected ReferenceInvoiceNumber INV-2026-103, got %s", alert.ReferenceInvoiceNumber)
		}
	})

	t.Run("Hike 2 (Fortune Sunflower Oil): 135 -> 145 (+7.4%)", func(t *testing.T) {
		newItem := models.LineItem{
			Description: "Fortune Sunlite Sunflower Oil 1L",
			PackSize:    "1l",
			UnitPrice:   145.00,
			Quantity:    10,
		}
		alerts := engine.CheckItemPrice(111, "INV-2026-111", "bill_111.jpg", now, "Kailash Agro Foods", newItem, history)
		if len(alerts) != 1 {
			t.Fatalf("Expected 1 alert for Fortune oil hike, got %d", len(alerts))
		}
		if alerts[0].ReferenceInvoiceNumber != "INV-2026-110" {
			t.Errorf("Expected reference bill INV-2026-110, got %s", alerts[0].ReferenceInvoiceNumber)
		}
	})

	t.Run("Hike 3 (Aashirvaad Atta): 380 -> 410 (+7.8%)", func(t *testing.T) {
		newItem := models.LineItem{
			Description: "Aashirvaad Shudh Chakki Atta 10kg",
			PackSize:    "10kg",
			UnitPrice:   410.00,
			Quantity:    5,
		}
		alerts := engine.CheckItemPrice(121, "INV-2026-121", "bill_121.jpg", now, "Bharat Trading Co", newItem, history)
		if len(alerts) != 1 {
			t.Fatalf("Expected 1 alert for Atta hike, got %d", len(alerts))
		}
	})

	t.Run("Hike 4 (Surf Excel): 145 -> 155 (+6.9%)", func(t *testing.T) {
		newItem := models.LineItem{
			Description: "Surf Excel Quick Wash 1kg",
			PackSize:    "1kg",
			UnitPrice:   155.00,
			Quantity:    10,
		}
		alerts := engine.CheckItemPrice(131, "INV-2026-131", "bill_131.jpg", now, "Sharma Distributors", newItem, history)
		if len(alerts) != 1 {
			t.Fatalf("Expected 1 alert for Surf Excel hike, got %d", len(alerts))
		}
	})

	t.Run("Hike 5 (Colgate Strong Teeth): 110 -> 120 (+9.1%)", func(t *testing.T) {
		newItem := models.LineItem{
			Description: "Colgate Strong Teeth 200g",
			PackSize:    "200g",
			UnitPrice:   120.00,
			Quantity:    15,
		}
		alerts := engine.CheckItemPrice(141, "INV-2026-141", "bill_141.jpg", now, "Bharat Trading Co", newItem, history)
		if len(alerts) != 1 {
			t.Fatalf("Expected 1 alert for Colgate hike, got %d", len(alerts))
		}
	})

	t.Run("Cheaper Elsewhere (Tata Salt): 28 vs 24 (-14.3%)", func(t *testing.T) {
		newItem := models.LineItem{
			Description:  "Tata Salt Vacuum Evaporated 1kg",
			PackSize:     "1kg",
			UnitPrice:    28.00,
			Quantity:     20,
			TaxableValue: 560.00,
		}

		alerts := engine.CheckItemPrice(105, "INV-2026-105", "bill_105.jpg", now, "Sharma Distributors", newItem, history)
		if len(alerts) != 1 {
			t.Fatalf("Expected exactly 1 alert, got %d", len(alerts))
		}
		if alerts[0].Type != "cheaper_elsewhere" {
			t.Errorf("Expected cheaper_elsewhere alert")
		}
		if alerts[0].ReferenceSupplier != "Bharat Trading Co" {
			t.Errorf("Expected reference supplier Bharat Trading Co, got %s", alerts[0].ReferenceSupplier)
		}
	})

	t.Run("No Alert on Normal Price / Identical Price", func(t *testing.T) {
		newItem := models.LineItem{
			Description:  "Parle-G 12-pack",
			PackSize:     "12-pack",
			UnitPrice:    118.00, // same price
			Quantity:     10,
			TaxableValue: 1180.00,
		}

		alerts := engine.CheckItemPrice(106, "INV-2026-106", "bill_106.jpg", now, "Sharma Distributors", newItem, history)
		if len(alerts) > 0 {
			t.Errorf("Expected 0 alerts for unchanged price, got %d", len(alerts))
		}
	})
}
