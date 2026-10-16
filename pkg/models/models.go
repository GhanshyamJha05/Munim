package models

import "time"

// LineItem represents an individual purchased line item on an invoice.
type LineItem struct {
	Description          string   `json:"description"`
	DescriptionNormalized string  `json:"description_normalized"`
	HSNCode              string   `json:"hsn_code"`
	PackSize             string   `json:"pack_size"`
	Unit                 string   `json:"unit"`
	Quantity             float64  `json:"quantity"`
	UnitPrice            float64  `json:"unit_price"`
	TaxableValue         float64  `json:"taxable_value"`
	GSTRate              float64  `json:"gst_rate"`
	Confidence           *float64 `json:"confidence,omitempty"`
}

// InvoiceExtraction represents the direct structured output from the vision LLM.
type InvoiceExtraction struct {
	VendorName       string     `json:"vendor_name"`
	VendorGSTIN      string     `json:"vendor_gstin"`
	InvoiceNumber    string     `json:"invoice_number"`
	InvoiceDate      string     `json:"invoice_date"` // Expected ISO 8601 (YYYY-MM-DD)
	DueDate          *string    `json:"due_date"`     // Nullable
	LineItems        []LineItem `json:"line_items"`
	Subtotal         float64    `json:"subtotal"`
	CGST             float64    `json:"cgst"`
	SGST             float64    `json:"sgst"`
	IGST             float64    `json:"igst"`
	RoundOff         float64    `json:"round_off"`
	Total            float64    `json:"total"`
	LanguageDetected string     `json:"language_detected"`
	Confidence       float64    `json:"confidence"` // Aggregate confidence 0.0 - 1.0
}

// BillValidationStatus indicates the verification state of a bill.
type BillValidationStatus string

const (
	StatusVerified    BillValidationStatus = "verified"
	StatusNeedsReview BillValidationStatus = "needs_review"
	StatusApproved    BillValidationStatus = "approved"
)

// BillRecord represents a stored bill in the SQLite database.
type BillRecord struct {
	ID               int64                `json:"id"`
	VendorName       string               `json:"vendor_name"`
	VendorGSTIN      string               `json:"vendor_gstin"`
	InvoiceNumber    string               `json:"invoice_number"`
	InvoiceDate      time.Time            `json:"invoice_date"`
	DueDate          *time.Time           `json:"due_date"`
	Subtotal         float64              `json:"subtotal"`
	CGST             float64              `json:"cgst"`
	SGST             float64              `json:"sgst"`
	IGST             float64              `json:"igst"`
	RoundOff         float64              `json:"round_off"`
	Total            float64              `json:"total"`
	ImagePath        string               `json:"image_path"`
	ValidationStatus BillValidationStatus `json:"validation_status"`
	ValidationErrors []string             `json:"validation_errors"`
	Confidence       float64              `json:"confidence"`
	LanguageDetected string               `json:"language_detected"`
	LineItems        []LineItem           `json:"line_items"`
	AmountPaid       float64              `json:"amount_paid"`
	PaymentStatus    string               `json:"payment_status"` // unpaid, partially_paid, paid
	ImageHash        string               `json:"image_hash,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
}

// PriceAlert represents a detected price hike or cheaper-elsewhere opportunity.
type PriceAlert struct {
	ID                     int64     `json:"id"`
	Type                   string    `json:"type"` // "spike_both", "spike_prior", "spike_median", "cheaper_elsewhere"
	ItemDescription        string    `json:"item_description"`
	ItemNormalized         string    `json:"item_normalized"`
	PackSize               string    `json:"pack_size"`
	CurrentBillID          int64     `json:"current_bill_id"`
	CurrentInvoiceNumber   string    `json:"current_invoice_number"`
	CurrentImagePath       string    `json:"current_image_path"`
	CurrentBillDate        time.Time `json:"current_bill_date"`
	CurrentSupplier        string    `json:"current_supplier"`
	CurrentPrice           float64   `json:"current_price"`
	ReferenceBillID        int64     `json:"reference_bill_id"`
	ReferenceInvoiceNumber string    `json:"reference_invoice_number"`
	ReferenceImagePath     string    `json:"reference_image_path"`
	ReferenceBillDate      time.Time `json:"reference_bill_date"`
	ReferenceSupplier      string    `json:"reference_supplier"`
	ReferencePrice         float64   `json:"reference_price"`
	PercentageDiff         float64   `json:"percentage_diff"`
	Message                string    `json:"message"`
	CreatedAt              time.Time `json:"created_at"`
}
