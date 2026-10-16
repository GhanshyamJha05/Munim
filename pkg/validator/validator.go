package validator

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/munim-app/munim/pkg/models"
)

// GSTINRegex validates the 15-character standard GSTIN structure in India.
// Structure: 2 digits (State) + 5 uppercase letters (PAN) + 4 digits + 1 letter + 1 digit/letter (entity #) + 'Z' + 1 checksum char
var GSTINRegex = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)

const gstinChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// ComputeGSTINChecksum returns the 15th checksum character for the first 14 characters of a GSTIN.
func ComputeGSTINChecksum(first14 string) string {
	first14 = strings.ToUpper(strings.TrimSpace(first14))
	if len(first14) < 14 {
		return ""
	}
	first14 = first14[:14]

	var sum int
	for i := 0; i < 14; i++ {
		ch := first14[i]
		idx := strings.IndexByte(gstinChars, ch)
		if idx == -1 {
			return ""
		}

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

// ValidateGSTIN verifies both the regex structure and the Luhn mod-36 checksum character.
func ValidateGSTIN(gstin string) (bool, string) {
	gstin = strings.TrimSpace(strings.ToUpper(gstin))
	if gstin == "" {
		return false, "GSTIN is missing"
	}

	if !GSTINRegex.MatchString(gstin) {
		return false, fmt.Sprintf("GSTIN %q does not match standard 15-character format", gstin)
	}

	expectedChar := ComputeGSTINChecksum(gstin[:14])
	if expectedChar == "" {
		return false, "Invalid characters in GSTIN"
	}
	actualChar := string(gstin[14])

	if actualChar != expectedChar {
		return false, fmt.Sprintf("GSTIN checksum mismatch: got %s, expected %s", actualChar, expectedChar)
	}

	return true, ""
}

// PlainError contains both user-friendly English and Hindi plain-language error explanations,
// along with the specific field that failed validation for UI highlighting.
type PlainError struct {
	Field   string `json:"field"` // e.g. "vendor_gstin", "subtotal", "total", "cgst", "invoice_date", "confidence"
	English string `json:"english"`
	Hindi   string `json:"hindi"`
}

// ValidationResult summarizes the verification outcome.
type ValidationResult struct {
	IsValid          bool         `json:"is_valid"`
	NeedsReview      bool         `json:"needs_review"`
	Errors           []string     `json:"errors"`
	PlainErrors      []PlainError `json:"plain_errors"`
	ConfidencePass   bool         `json:"confidence_pass"`
	RepairSuggestion string       `json:"repair_suggestion,omitempty"`
}

// ValidateInvoice performs deterministic checks on an extracted invoice:
// 1. Valid GSTIN & checksum
// 2. Line items sum to subtotal within Rs 1.00
// 3. Subtotal + taxes + round_off == Total within Rs 1.00
// 4. Intra-state equivalence (CGST == SGST if IGST is zero)
// 5. Valid parseable invoice date
// 6. Overall confidence >= 0.70
func ValidateInvoice(inv *models.InvoiceExtraction) *ValidationResult {
	res := &ValidationResult{
		IsValid:        true,
		NeedsReview:    false,
		Errors:         []string{},
		PlainErrors:    []PlainError{},
		ConfidencePass: true,
	}

	// 1. GSTIN Check
	if ok, errMsg := ValidateGSTIN(inv.VendorGSTIN); !ok {
		res.Errors = append(res.Errors, errMsg)
		res.PlainErrors = append(res.PlainErrors, PlainError{
			Field:   "vendor_gstin",
			English: fmt.Sprintf("Supplier GSTIN (%s) is invalid or checksum digit does not match.", inv.VendorGSTIN),
			Hindi:   fmt.Sprintf("सप्लायर का GSTIN (%s) अमान्य है या चेक-डिजिट मेल नहीं खा रहा है।", inv.VendorGSTIN),
		})
	}

	// 2. Line Items vs Subtotal
	var itemsSum float64
	for _, item := range inv.LineItems {
		itemsSum += item.TaxableValue
	}
	if len(inv.LineItems) == 0 {
		res.Errors = append(res.Errors, "No line items extracted from bill")
		res.PlainErrors = append(res.PlainErrors, PlainError{
			Field:   "line_items",
			English: "No items could be read from this bill.",
			Hindi:   "इस बिल से कोई सामान पढ़ा नहीं जा सका।",
		})
	} else if math.Abs(itemsSum-inv.Subtotal) > 1.00 {
		diff := math.Abs(itemsSum - inv.Subtotal)
		res.Errors = append(res.Errors, fmt.Sprintf(
			"Line items taxable total (%.2f) does not match subtotal (%.2f), diff: %.2f",
			itemsSum, inv.Subtotal, diff,
		))
		res.PlainErrors = append(res.PlainErrors, PlainError{
			Field:   "subtotal",
			English: fmt.Sprintf("Line items add up to ₹%.2f, but subtotal says ₹%.2f (difference of ₹%.2f).", itemsSum, inv.Subtotal, diff),
			Hindi:   fmt.Sprintf("सामानों का कुल जोड़ ₹%.2f है, जबकि सब-टोटल ₹%.2f लिखा है (₹%.2f का अंतर)।", itemsSum, inv.Subtotal, diff),
		})
	}

	// 3. Subtotal + Taxes + RoundOff == Total
	expectedTotal := inv.Subtotal + inv.CGST + inv.SGST + inv.IGST + inv.RoundOff
	if math.Abs(expectedTotal-inv.Total) > 1.00 {
		diff := math.Abs(expectedTotal - inv.Total)
		res.Errors = append(res.Errors, fmt.Sprintf(
			"Subtotal (%.2f) + CGST (%.2f) + SGST (%.2f) + IGST (%.2f) + RoundOff (%.2f) = %.2f, does not match Grand Total (%.2f)",
			inv.Subtotal, inv.CGST, inv.SGST, inv.IGST, inv.RoundOff, expectedTotal, inv.Total,
		))
		res.PlainErrors = append(res.PlainErrors, PlainError{
			Field:   "total",
			English: fmt.Sprintf("Calculated total with taxes is ₹%.2f, but printed grand total is ₹%.2f (difference of ₹%.2f).", expectedTotal, inv.Total, diff),
			Hindi:   fmt.Sprintf("टैक्स जोड़कर कुल ₹%.2f बनता है, लेकिन छपा हुआ ग्रैंड टोटल ₹%.2f है (₹%.2f का अंतर)।", expectedTotal, inv.Total, diff),
		})
	}

	// 4. Intra-state CGST == SGST
	if inv.IGST == 0 && (inv.CGST > 0 || inv.SGST > 0) {
		if math.Abs(inv.CGST-inv.SGST) > 0.05 {
			res.Errors = append(res.Errors, fmt.Sprintf(
				"Intra-state tax mismatch: CGST (%.2f) does not equal SGST (%.2f)",
				inv.CGST, inv.SGST,
			))
			res.PlainErrors = append(res.PlainErrors, PlainError{
				Field:   "cgst",
				English: fmt.Sprintf("State and Central tax mismatch: CGST is ₹%.2f but SGST is ₹%.2f.", inv.CGST, inv.SGST),
				Hindi:   fmt.Sprintf("लोकल टैक्स में अंतर: CGST ₹%.2f है जबकि SGST ₹%.2f लिखा है।", inv.CGST, inv.SGST),
			})
		}
	}

	// 5. Date validation
	parsedDate, err := ParseDate(inv.InvoiceDate)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("Invalid invoice date %q: %v", inv.InvoiceDate, err))
		res.PlainErrors = append(res.PlainErrors, PlainError{
			Field:   "invoice_date",
			English: fmt.Sprintf("Invoice date (%s) is invalid or could not be understood.", inv.InvoiceDate),
			Hindi:   fmt.Sprintf("बिल की तारीख (%s) अमान्य है या समझ नहीं आई।", inv.InvoiceDate),
		})
	} else {
		// Cannot be more than 1 day in the future
		if parsedDate.After(time.Now().Add(24 * time.Hour)) {
			res.Errors = append(res.Errors, fmt.Sprintf("Invoice date %s is in the future", inv.InvoiceDate))
			res.PlainErrors = append(res.PlainErrors, PlainError{
				Field:   "invoice_date",
				English: fmt.Sprintf("Invoice date (%s) is set in the future.", inv.InvoiceDate),
				Hindi:   fmt.Sprintf("बिल की तारीख (%s) भविष्य की है।", inv.InvoiceDate),
			})
		}
	}

	// 6. Confidence threshold
	if inv.Confidence < 0.70 {
		res.ConfidencePass = false
		res.Errors = append(res.Errors, fmt.Sprintf("Extraction confidence %.2f is below threshold 0.70", inv.Confidence))
		res.PlainErrors = append(res.PlainErrors, PlainError{
			Field:   "confidence",
			English: fmt.Sprintf("Bill photo was blurry, faded, or handwritten (legibility rating: %.0f%%). Please verify values.", inv.Confidence*100),
			Hindi:   fmt.Sprintf("बिल की फोटो धुंधली, फीकी या हाथ से लिखी हुई थी (स्पष्टता: %.0f%%)। कृपया आंकड़े जांच लें।", inv.Confidence*100),
		})
	}

	if len(res.Errors) > 0 {
		res.IsValid = false
		res.NeedsReview = true
		res.RepairSuggestion = strings.Join(res.Errors, "; ")
	}

	return res
}

// ParseDate handles multiple common Indian invoice date formats.
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"02.01.2006",
		"2006/01/02",
		"2 Jan 2006",
		"02 Jan 2006",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date %q", s)
}
