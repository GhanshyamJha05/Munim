package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/munim-app/munim/pkg/models"
	"github.com/munim-app/munim/pkg/preproc"
	"github.com/munim-app/munim/pkg/pricealert"
)

// Store encapsulates SQLite database operations.
type Store struct {
	db *sql.DB
}

// Open initializes SQLite file database with schema migrations.
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil && filepath.Dir(dbPath) != "." {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Optimize SQLite for concurrent single-binary local-first usage
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS bills (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_name TEXT NOT NULL,
		vendor_gstin TEXT NOT NULL,
		invoice_number TEXT NOT NULL,
		invoice_date TEXT NOT NULL,
		due_date TEXT,
		subtotal REAL NOT NULL,
		cgst REAL NOT NULL,
		sgst REAL NOT NULL,
		igst REAL NOT NULL,
		round_off REAL NOT NULL,
		total REAL NOT NULL,
		image_path TEXT NOT NULL,
		validation_status TEXT NOT NULL,
		validation_errors TEXT,
		confidence REAL NOT NULL,
		language_detected TEXT,
		line_items_json TEXT NOT NULL,
		amount_paid REAL NOT NULL DEFAULT 0.0,
		payment_status TEXT NOT NULL DEFAULT 'unpaid',
		image_hash TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(vendor_name, invoice_number, invoice_date)
	);

	CREATE TABLE IF NOT EXISTS price_alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		item_description TEXT NOT NULL,
		item_normalized TEXT NOT NULL,
		pack_size TEXT NOT NULL,
		current_bill_id INTEGER NOT NULL,
		current_invoice_number TEXT,
		current_image_path TEXT,
		current_bill_date TEXT NOT NULL,
		current_supplier TEXT NOT NULL,
		current_price REAL NOT NULL,
		reference_bill_id INTEGER NOT NULL,
		reference_invoice_number TEXT,
		reference_image_path TEXT,
		reference_bill_date TEXT NOT NULL,
		reference_supplier TEXT NOT NULL,
		reference_price REAL NOT NULL,
		percentage_diff REAL NOT NULL,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_bills_vendor ON bills(vendor_name);
	CREATE INDEX IF NOT EXISTS idx_bills_date ON bills(invoice_date);
	CREATE INDEX IF NOT EXISTS idx_alerts_current ON price_alerts(current_bill_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// InsertBill records a newly processed bill and returns its ID.
func (s *Store) InsertBill(ctx context.Context, bill *models.BillRecord) (int64, error) {
	itemsJSON, err := json.Marshal(bill.LineItems)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal line items: %w", err)
	}

	errsJSON, _ := json.Marshal(bill.ValidationErrors)

	var dueDateStr *string
	if bill.DueDate != nil {
		d := bill.DueDate.Format("2006-01-02")
		dueDateStr = &d
	}

	query := `
	INSERT INTO bills (
		vendor_name, vendor_gstin, invoice_number, invoice_date, due_date,
		subtotal, cgst, sgst, igst, round_off, total,
		image_path, validation_status, validation_errors, confidence,
		language_detected, line_items_json, amount_paid, payment_status, image_hash, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := s.db.ExecContext(ctx, query,
		bill.VendorName,
		bill.VendorGSTIN,
		bill.InvoiceNumber,
		bill.InvoiceDate.Format("2006-01-02"),
		dueDateStr,
		bill.Subtotal,
		bill.CGST,
		bill.SGST,
		bill.IGST,
		bill.RoundOff,
		bill.Total,
		bill.ImagePath,
		string(bill.ValidationStatus),
		string(errsJSON),
		bill.Confidence,
		bill.LanguageDetected,
		string(itemsJSON),
		bill.AmountPaid,
		bill.PaymentStatus,
		bill.ImageHash,
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	bill.ID = id
	return id, err
}

// FindDuplicateBill checks if an existing bill matches by (supplier + invoice_number + invoice_date) or near-identical image perceptual hash.
func (s *Store) FindDuplicateBill(ctx context.Context, vendorName, invoiceNumber, invoiceDate, imageHash string) (*models.BillRecord, string, error) {
	bills, err := s.GetAllBills(ctx)
	if err != nil {
		return nil, "", err
	}

	vNorm := strings.ToLower(strings.TrimSpace(vendorName))
	invNorm := strings.ToLower(strings.TrimSpace(invoiceNumber))

	for _, b := range bills {
		bVNorm := strings.ToLower(strings.TrimSpace(b.VendorName))
		bInvNorm := strings.ToLower(strings.TrimSpace(b.InvoiceNumber))
		bDateStr := b.InvoiceDate.Format("2006-01-02")

		// 1. Exact match on supplier + invoice_number + date
		if vNorm != "" && invNorm != "" && bVNorm == vNorm && bInvNorm == invNorm && bDateStr == invoiceDate {
			return &b, fmt.Sprintf("Supplier %q, Invoice #%s and Date %s already exists", b.VendorName, b.InvoiceNumber, bDateStr), nil
		}

		// 2. Perceptual image hash near-identical match (Hamming distance <= 6)
		if imageHash != "" && b.ImageHash != "" {
			dist := preproc.HammingDistance(imageHash, b.ImageHash)
			if dist <= 6 {
				return &b, fmt.Sprintf("Bill image appears visually identical to existing Invoice #%s (Hamming distance %d)", b.InvoiceNumber, dist), nil
			}
		}
	}

	return nil, "", nil
}

// UpdateBillDetails allows full editing of supplier, GSTIN, invoice number, date, taxes, round-off, items and total.
func (s *Store) UpdateBillDetails(ctx context.Context, bill *models.BillRecord) error {
	itemsJSON, err := json.Marshal(bill.LineItems)
	if err != nil {
		return err
	}
	errsJSON, _ := json.Marshal(bill.ValidationErrors)

	var dueDateStr *string
	if bill.DueDate != nil {
		d := bill.DueDate.Format("2006-01-02")
		dueDateStr = &d
	}

	query := `
	UPDATE bills SET
		vendor_name = ?, vendor_gstin = ?, invoice_number = ?, invoice_date = ?, due_date = ?,
		subtotal = ?, cgst = ?, sgst = ?, igst = ?, round_off = ?, total = ?,
		validation_status = ?, validation_errors = ?, line_items_json = ?
	WHERE id = ?`

	_, err = s.db.ExecContext(ctx, query,
		bill.VendorName,
		bill.VendorGSTIN,
		bill.InvoiceNumber,
		bill.InvoiceDate.Format("2006-01-02"),
		dueDateStr,
		bill.Subtotal,
		bill.CGST,
		bill.SGST,
		bill.IGST,
		bill.RoundOff,
		bill.Total,
		string(bill.ValidationStatus),
		string(errsJSON),
		string(itemsJSON),
		bill.ID,
	)
	return err
}

// UpdateBillStatusAndItems updates review edits made by the user.
func (s *Store) UpdateBillStatusAndItems(ctx context.Context, id int64, status models.BillValidationStatus, items []models.LineItem, total float64) error {
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return err
	}

	query := `UPDATE bills SET validation_status = ?, line_items_json = ?, total = ? WHERE id = ?`
	_, err = s.db.ExecContext(ctx, query, string(status), string(itemsJSON), total, id)
	return err
}

// RecordPayment updates amount paid and payment status for a bill.
func (s *Store) RecordPayment(ctx context.Context, billID int64, amount float64) error {
	return s.RecordPaymentWithDetails(ctx, billID, amount, "", "")
}

// RecordPaymentWithDetails records amount paid with payment date and mode (cash/UPI/bank).
func (s *Store) RecordPaymentWithDetails(ctx context.Context, billID int64, amount float64, paymentDate string, paymentMode string) error {
	var total, currentPaid float64
	err := s.db.QueryRowContext(ctx, `SELECT total, amount_paid FROM bills WHERE id = ?`, billID).Scan(&total, &currentPaid)
	if err != nil {
		return err
	}

	newPaid := currentPaid + amount
	status := "partially_paid"
	if newPaid >= total-0.01 {
		status = "paid"
		newPaid = total
	}

	_, err = s.db.ExecContext(ctx, `UPDATE bills SET amount_paid = ?, payment_status = ? WHERE id = ?`, newPaid, status, billID)
	return err
}

// SavePriceAlert records an alert.
func (s *Store) SavePriceAlert(ctx context.Context, a models.PriceAlert) error {
	query := `
	INSERT INTO price_alerts (
		type, item_description, item_normalized, pack_size,
		current_bill_id, current_invoice_number, current_image_path, current_bill_date, current_supplier, current_price,
		reference_bill_id, reference_invoice_number, reference_image_path, reference_bill_date, reference_supplier, reference_price,
		percentage_diff, message, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		a.Type,
		a.ItemDescription,
		a.ItemNormalized,
		a.PackSize,
		a.CurrentBillID,
		a.CurrentInvoiceNumber,
		a.CurrentImagePath,
		a.CurrentBillDate.Format("2006-01-02"),
		a.CurrentSupplier,
		a.CurrentPrice,
		a.ReferenceBillID,
		a.ReferenceInvoiceNumber,
		a.ReferenceImagePath,
		a.ReferenceBillDate.Format("2006-01-02"),
		a.ReferenceSupplier,
		a.ReferencePrice,
		a.PercentageDiff,
		a.Message,
		time.Now().Format(time.RFC3339),
	)
	return err
}

// GetAllBills returns all bills ordered by invoice date descending.
func (s *Store) GetAllBills(ctx context.Context) ([]models.BillRecord, error) {
	query := `
	SELECT id, vendor_name, vendor_gstin, invoice_number, invoice_date, due_date,
	       subtotal, cgst, sgst, igst, round_off, total, image_path,
	       validation_status, validation_errors, confidence, language_detected,
	       line_items_json, amount_paid, payment_status, image_hash, created_at
	FROM bills
	ORDER BY invoice_date DESC, id DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bills []models.BillRecord
	for rows.Next() {
		var b models.BillRecord
		var invDateStr, createdAtStr string
		var dueDateStr, errsJSON, itemsJSON, imageHashStr *string

		err := rows.Scan(
			&b.ID, &b.VendorName, &b.VendorGSTIN, &b.InvoiceNumber, &invDateStr, &dueDateStr,
			&b.Subtotal, &b.CGST, &b.SGST, &b.IGST, &b.RoundOff, &b.Total, &b.ImagePath,
			&b.ValidationStatus, &errsJSON, &b.Confidence, &b.LanguageDetected,
			&itemsJSON, &b.AmountPaid, &b.PaymentStatus, &imageHashStr, &createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		b.InvoiceDate, _ = time.Parse("2006-01-02", invDateStr)
		if dueDateStr != nil && *dueDateStr != "" {
			d, err := time.Parse("2006-01-02", *dueDateStr)
			if err == nil {
				b.DueDate = &d
			}
		}
		if errsJSON != nil && *errsJSON != "" {
			_ = json.Unmarshal([]byte(*errsJSON), &b.ValidationErrors)
		}
		if itemsJSON != nil && *itemsJSON != "" {
			_ = json.Unmarshal([]byte(*itemsJSON), &b.LineItems)
		}
		if imageHashStr != nil {
			b.ImageHash = *imageHashStr
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		bills = append(bills, b)
	}

	return bills, nil
}

// GetHistoricalPurchases returns past line item purchases for price matching.
func (s *Store) GetHistoricalPurchases(ctx context.Context) ([]pricealert.HistoricalPurchase, error) {
	bills, err := s.GetAllBills(ctx)
	if err != nil {
		return nil, err
	}

	var history []pricealert.HistoricalPurchase
	for _, b := range bills {
		for _, it := range b.LineItems {
			history = append(history, pricealert.HistoricalPurchase{
				BillID:        b.ID,
				InvoiceNumber: b.InvoiceNumber,
				ImagePath:     b.ImagePath,
				BillDate:      b.InvoiceDate,
				SupplierName:  b.VendorName,
				ItemDesc:      it.Description,
				PackSize:      it.PackSize,
				HSNCode:       it.HSNCode,
				UnitPrice:     it.UnitPrice,
				Quantity:      it.Quantity,
			})
		}
	}

	return history, nil
}

// GetAllPriceAlerts returns all price alerts.
func (s *Store) GetAllPriceAlerts(ctx context.Context) ([]models.PriceAlert, error) {
	query := `
	SELECT id, type, item_description, item_normalized, pack_size,
	       current_bill_id, current_invoice_number, current_image_path, current_bill_date, current_supplier, current_price,
	       reference_bill_id, reference_invoice_number, reference_image_path, reference_bill_date, reference_supplier, reference_price,
	       percentage_diff, message, created_at
	FROM price_alerts
	ORDER BY current_bill_date DESC, id DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.PriceAlert
	for rows.Next() {
		var a models.PriceAlert
		var curDateStr, refDateStr, createdStr string
		var curInv, curImg, refInv, refImg *string

		err := rows.Scan(
			&a.ID, &a.Type, &a.ItemDescription, &a.ItemNormalized, &a.PackSize,
			&a.CurrentBillID, &curInv, &curImg, &curDateStr, &a.CurrentSupplier, &a.CurrentPrice,
			&a.ReferenceBillID, &refInv, &refImg, &refDateStr, &a.ReferenceSupplier, &a.ReferencePrice,
			&a.PercentageDiff, &a.Message, &createdStr,
		)
		if err != nil {
			return nil, err
		}

		if curInv != nil {
			a.CurrentInvoiceNumber = *curInv
		}
		if curImg != nil {
			a.CurrentImagePath = *curImg
		}
		if refInv != nil {
			a.ReferenceInvoiceNumber = *refInv
		}
		if refImg != nil {
			a.ReferenceImagePath = *refImg
		}

		a.CurrentBillDate, _ = time.Parse("2006-01-02", curDateStr)
		a.ReferenceBillDate, _ = time.Parse("2006-01-02", refDateStr)
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)

		alerts = append(alerts, a)
	}

	return alerts, nil
}

// SupplierDueSummary computes outstanding amounts and overdue bills per supplier.
type SupplierDueSummary struct {
	SupplierName          string    `json:"supplier_name"`
	TotalBilled           float64   `json:"total_billed"`
	TotalPaid             float64   `json:"total_paid"`
	BalanceDue            float64   `json:"balance_due"`
	EarliestDue           time.Time `json:"earliest_due"`
	EarliestBillID        int64     `json:"earliest_bill_id"`
	EarliestInvoiceNumber string    `json:"earliest_invoice_number"`
	IsOverdue             bool      `json:"is_overdue"`
	OverdueBills          int       `json:"overdue_bills"`
	OverdueDays           int       `json:"overdue_days"`
}

// GetSupplierDues aggregates ledger balances and tracks overdues based on the real current date.
func (s *Store) GetSupplierDues(ctx context.Context) ([]SupplierDueSummary, error) {
	bills, err := s.GetAllBills(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	summaryMap := make(map[string]*SupplierDueSummary)

	for _, b := range bills {
		sum, exists := summaryMap[b.VendorName]
		if !exists {
			sum = &SupplierDueSummary{
				SupplierName: b.VendorName,
			}
			summaryMap[b.VendorName] = sum
		}

		sum.TotalBilled += b.Total
		sum.TotalPaid += b.AmountPaid
		balance := b.Total - b.AmountPaid
		sum.BalanceDue += balance

		if balance > 0.01 {
			// Compute effective due date: if bill due_date is nil, default to invoice_date + 30 days
			dueDate := b.InvoiceDate.AddDate(0, 0, 30)
			if b.DueDate != nil {
				dueDate = *b.DueDate
			}

			if sum.EarliestDue.IsZero() || dueDate.Before(sum.EarliestDue) {
				sum.EarliestDue = dueDate
				sum.EarliestBillID = b.ID
				sum.EarliestInvoiceNumber = b.InvoiceNumber
			}

			if now.After(dueDate) {
				sum.IsOverdue = true
				sum.OverdueBills++
				days := int(now.Sub(dueDate).Hours() / 24)
				if days > sum.OverdueDays {
					sum.OverdueDays = days
				}
			}
		}
	}

	var summaries []SupplierDueSummary
	for _, sum := range summaryMap {
		summaries = append(summaries, *sum)
	}

	return summaries, nil
}
