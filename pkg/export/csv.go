package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/munim-app/munim/pkg/models"
)

// GenerateCAPack generates a comprehensive CSV with one row per line item and invoice headers repeated.
func GenerateCAPack(bills []models.BillRecord, filterMonth time.Time) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	headers := []string{
		"Invoice Number",
		"Invoice Date",
		"Supplier Name",
		"Supplier GSTIN",
		"HSN Code",
		"Item Description",
		"Pack Size",
		"Quantity",
		"Unit",
		"Unit Price (₹)",
		"Taxable Value (₹)",
		"GST Rate (%)",
		"CGST (₹)",
		"SGST (₹)",
		"IGST (₹)",
		"Round Off (₹)",
		"Invoice Total (₹)",
		"Validation Status",
		"Payment Status",
	}

	if err := w.Write(headers); err != nil {
		return nil, err
	}

	for _, b := range bills {
		if !filterMonth.IsZero() {
			if b.InvoiceDate.Year() != filterMonth.Year() || b.InvoiceDate.Month() != filterMonth.Month() {
				continue
			}
		}

		for i, item := range b.LineItems {
			// Print bill-level taxes and totals only on the first line item to prevent CA sum multiplication
			cgstStr := ""
			sgstStr := ""
			igstStr := ""
			roundOffStr := ""
			totalStr := ""
			if i == 0 {
				cgstStr = fmt.Sprintf("%.2f", b.CGST)
				sgstStr = fmt.Sprintf("%.2f", b.SGST)
				igstStr = fmt.Sprintf("%.2f", b.IGST)
				roundOffStr = fmt.Sprintf("%.2f", b.RoundOff)
				totalStr = fmt.Sprintf("%.2f", b.Total)
			}

			row := []string{
				b.InvoiceNumber,
				b.InvoiceDate.Format("2006-01-02"),
				b.VendorName,
				b.VendorGSTIN,
				item.HSNCode,
				item.Description,
				item.PackSize,
				fmt.Sprintf("%.2f", item.Quantity),
				item.Unit,
				fmt.Sprintf("%.2f", item.UnitPrice),
				fmt.Sprintf("%.2f", item.TaxableValue),
				fmt.Sprintf("%.1f", item.GSTRate),
				cgstStr,
				sgstStr,
				igstStr,
				roundOffStr,
				totalStr,
				string(b.ValidationStatus),
				b.PaymentStatus,
			}
			if err := w.Write(row); err != nil {
				return nil, err
			}
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// GenerateTallyVouchers generates a purchase-voucher CSV ready to import into Tally ERP/Prime.
func GenerateTallyVouchers(bills []models.BillRecord, filterMonth time.Time) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	headers := []string{
		"Vch Date",
		"Vch Type",
		"Vch No",
		"Party Ledger Name",
		"Purchase Ledger",
		"Item Name",
		"Item Qty",
		"Item Rate",
		"Item Amount",
		"Total CGST",
		"Total SGST",
		"Total IGST",
		"Total Invoice Amount",
		"Narration",
	}

	if err := w.Write(headers); err != nil {
		return nil, err
	}

	for _, b := range bills {
		if !filterMonth.IsZero() {
			if b.InvoiceDate.Year() != filterMonth.Year() || b.InvoiceDate.Month() != filterMonth.Month() {
				continue
			}
		}

		narration := fmt.Sprintf("GSTIN: %s | Status: %s", b.VendorGSTIN, b.ValidationStatus)

		if len(b.LineItems) == 0 {
			// Single voucher row without line items
			row := []string{
				b.InvoiceDate.Format("02-01-2006"),
				"Purchase",
				b.InvoiceNumber,
				b.VendorName,
				"Local Purchase @ 18%",
				"General Goods",
				"1",
				fmt.Sprintf("%.2f", b.Subtotal),
				fmt.Sprintf("%.2f", b.Subtotal),
				fmt.Sprintf("%.2f", b.CGST),
				fmt.Sprintf("%.2f", b.SGST),
				fmt.Sprintf("%.2f", b.IGST),
				fmt.Sprintf("%.2f", b.Total),
				narration,
			}
			_ = w.Write(row)
			continue
		}

		for i, item := range b.LineItems {
			taxName := "Local Purchase GST"
			if b.IGST > 0 {
				taxName = "Interstate Purchase IGST"
			}

			// In multi-item vouchers, invoice taxes and grand total belong on the primary voucher entry to avoid inflation
			cgstStr := ""
			sgstStr := ""
			igstStr := ""
			totalStr := ""
			if i == 0 {
				cgstStr = fmt.Sprintf("%.2f", b.CGST)
				sgstStr = fmt.Sprintf("%.2f", b.SGST)
				igstStr = fmt.Sprintf("%.2f", b.IGST)
				totalStr = fmt.Sprintf("%.2f", b.Total)
			}

			row := []string{
				b.InvoiceDate.Format("02-01-2006"),
				"Purchase",
				b.InvoiceNumber,
				b.VendorName,
				taxName,
				item.Description,
				fmt.Sprintf("%.2f", item.Quantity),
				fmt.Sprintf("%.2f", item.UnitPrice),
				fmt.Sprintf("%.2f", item.TaxableValue),
				cgstStr,
				sgstStr,
				igstStr,
				totalStr,
				narration,
			}
			_ = w.Write(row)
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
