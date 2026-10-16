package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/munim-app/munim/pkg/config"
	"github.com/munim-app/munim/pkg/models"
	"github.com/munim-app/munim/pkg/validator"
)

// GroundTruthBill wraps invoice extraction data with ground truth expectations and deliberate challenge tags.
type GroundTruthBill struct {
	BillID            int64                    `json:"bill_id"`
	FileName          string                   `json:"file_name"`
	Distributor       string                   `json:"distributor"`
	ExpectedData      models.InvoiceExtraction `json:"expected_data"`
	DeliberateAnomaly string                   `json:"deliberate_anomaly,omitempty"` // e.g. "invalid_gstin", "bad_math", "low_confidence_total"
	ExpectedAlerts    []string                 `json:"expected_alerts,omitempty"`
}

type Distributor struct {
	Name     string
	Base14   string
	GSTIN    string
	City     string
	Phone    string
}

func getDistributors() []Distributor {
	return []Distributor{
		{
			Name:   "Sharma Distributors",
			Base14: "07AAAAA0000A1Z",
			GSTIN:  "07AAAAA0000A1Z" + validator.ComputeGSTINChecksum("07AAAAA0000A1Z"),
			City:   "Delhi",
			Phone:  "9810012345",
		},
		{
			Name:   "Bharat Trading Co",
			Base14: "07BBBCC1111B1Z",
			GSTIN:  "07BBBCC1111B1Z" + validator.ComputeGSTINChecksum("07BBBCC1111B1Z"),
			City:   "Delhi",
			Phone:  "9820054321",
		},
		{
			Name:   "Kailash Agro Foods",
			Base14: "07CCCDD2222C1Z",
			GSTIN:  "07CCCDD2222C1Z" + validator.ComputeGSTINChecksum("07CCCDD2222C1Z"),
			City:   "Delhi",
			Phone:  "9830098765",
		},
	}
}

func main() {
	outDir := "demo_data"
	imgDir := filepath.Join(outDir, "bills")
	_ = os.MkdirAll(imgDir, 0755)

	log.Printf("Generating 30 realistic synthetic bills for 'Gupta General Store' into %s...", outDir)

	// Requirement 5: Generate demo dates relative to today spread over last 60 days
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	startDate := today.AddDate(0, 0, -58)

	var allTruths []GroundTruthBill
	billCounter := 1
	dists := getDistributors()
	schedule := config.OfficialKiranaGSTSchedule

	// Generate 30 bills spread across last 60 days
	for i := 0; i < 30; i++ {
		// Evenly space 30 bills over ~58 days (2 days per bill)
		billDate := startDate.AddDate(0, 0, i*2)
		distIdx := i % len(dists)
		dist := dists[distIdx]
		invNum := fmt.Sprintf("INV-%d-%03d", billDate.Year(), billCounter)
		imgFileName := fmt.Sprintf("bill_%03d_%s.jpg", billCounter, strings.ReplaceAll(dist.Name, " ", "_"))

		// Select 2-4 items
		numItems := 2 + (i % 3)
		var items []models.LineItem
		var subtotal float64
		var totalTax float64
		anomaly := ""
		var expectedAlerts []string

		for j := 0; j < numItems; j++ {
			cat := schedule[(i+j)%len(schedule)]
			qty := float64(5 + (j * 5))
			rate := cat.BaseRate

			// SCRIPTED PRICE SPIKES & CHEAPER ELSEWHERE (All 5 Hikes + 1 Cheaper-Elsewhere)
			// Hike 1: Parle-G on bill #25 (index 24, Sharma Distributors) hike from 118.00 -> 123.00 (+4.2%)
			if i == 24 && cat.Description == "Parle-G 12-pack Biscuit" {
				rate = 123.00
				expectedAlerts = append(expectedAlerts, "Parle-G 12-pack is 4.2% higher than on your last bill from Sharma Distributors")
			}
			// Hike 2: Fortune Sunflower Oil on bill #24 (index 23, Kailash Agro Foods) hike from 135.00 -> 145.00 (+7.4%)
			if i == 23 && cat.Description == "Fortune Sunlite Sunflower Oil 1L" {
				rate = 145.00
				expectedAlerts = append(expectedAlerts, "Fortune Sunlite Sunflower Oil 1L is 7.4% higher than on your last bill")
			}
			// Hike 3: Atta on bill #26 (index 25, Bharat Trading Co) hike from 380.00 -> 410.00 (+7.8%)
			if i == 25 && cat.Description == "Aashirvaad Shudh Chakki Atta 10kg" {
				rate = 410.00
				expectedAlerts = append(expectedAlerts, "Aashirvaad Shudh Chakki Atta 10kg price spike")
			}
			// Hike 4: Surf Excel on bill #22 (index 21, Sharma Distributors) hike from 145.00 -> 155.00 (+6.9%)
			if i == 21 && cat.Description == "Surf Excel Quick Wash 1kg" {
				rate = 155.00
				expectedAlerts = append(expectedAlerts, "Surf Excel Quick Wash 1kg price spike")
			}
			// Hike 5: Colgate on bill #23 (index 22, Bharat Trading Co) hike from 110.00 -> 120.00 (+9.1%)
			if i == 22 && cat.Description == "Colgate Strong Teeth 200g" {
				rate = 120.00
				expectedAlerts = append(expectedAlerts, "Colgate Strong Teeth 200g price spike")
			}
			// Cheaper Elsewhere: Dettol Soap on bill #4 (index 3, Sharma Distributors). Sharma charges 44.00,
			// while Kailash Agro Foods sold it on bill #3 for 38.00 (save 13.6%), triggering Cheaper Elsewhere alert!
			if i == 3 && dist.Name == "Sharma Distributors" && cat.Description == "Dettol Original Soap 75g" {
				rate = 44.00
				expectedAlerts = append(expectedAlerts, "Dettol Original Soap 75g is available cheaper from Kailash Agro Foods")
			}

			taxable := qty * rate
			taxAmt := taxable * (cat.GSTRate / 100.0)

			items = append(items, models.LineItem{
				Description:  cat.Description,
				PackSize:     cat.PackSize,
				Unit:         cat.Unit,
				HSNCode:      cat.HSNCode,
				Quantity:     qty,
				UnitPrice:    rate,
				TaxableValue: taxable,
				GSTRate:      cat.GSTRate,
			})

			subtotal += taxable
			totalTax += taxAmt
		}

		cgst := totalTax / 2.0
		sgst := totalTax / 2.0
		grandTotal := subtotal + cgst + sgst

		// EXACTLY 3 DELIBERATE INVALID BILLS:
		// 1. One wrong GSTIN (Bill #8, index 7)
		// 2. One arithmetic mismatch (Bill #12, index 11)
		// 3. One low-confidence handwritten total (Bill #15, index 14)
		currentGSTIN := dist.GSTIN
		confidence := 0.95

		if i == 7 {
			// Deliberate invalid GSTIN checksum: flip last digit
			lastChar := dist.GSTIN[14]
			var wrongChar byte = '0'
			if lastChar == '0' {
				wrongChar = '1'
			}
			currentGSTIN = dist.GSTIN[:14] + string(wrongChar)
			anomaly = "invalid_gstin"
		} else if i == 11 {
			// Deliberate arithmetic math error
			grandTotal += 50.00
			anomaly = "bad_math"
		} else if i == 14 {
			// Deliberate low-confidence handwritten total
			confidence = 0.62 // below 0.70 threshold
			anomaly = "low_confidence_total"
		}

		// Cosmetic variations that REMAIN VALID (do not trigger needs-review)
		displayAnomaly := anomaly
		if i == 17 {
			displayAnomaly = "hindi_header"
		} else if i == 23 {
			displayAnomaly = "rotated_image"
		}

		dueDateStr := billDate.AddDate(0, 0, 30).Format("2006-01-02")
		extraction := models.InvoiceExtraction{
			VendorName:       dist.Name,
			VendorGSTIN:      currentGSTIN,
			InvoiceNumber:    invNum,
			InvoiceDate:      billDate.Format("2006-01-02"),
			DueDate:          &dueDateStr,
			LineItems:        items,
			Subtotal:         math.Round(subtotal*100) / 100,
			CGST:             math.Round(cgst*100) / 100,
			SGST:             math.Round(sgst*100) / 100,
			IGST:             0.0,
			RoundOff:         0.0,
			Total:            math.Round(grandTotal*100) / 100,
			LanguageDetected: "English",
			Confidence:       confidence,
		}

		if displayAnomaly == "hindi_header" {
			extraction.LanguageDetected = "Hinglish/Hindi"
		}

		// Render bill image
		renderBillImage(filepath.Join(imgDir, imgFileName), extraction, dist.City, dist.Phone, displayAnomaly)

		// Record ground truth
		allTruths = append(allTruths, GroundTruthBill{
			BillID:            int64(billCounter),
			FileName:          imgFileName,
			Distributor:       dist.Name,
			ExpectedData:      extraction,
			DeliberateAnomaly: anomaly,
			ExpectedAlerts:    expectedAlerts,
		})

		billCounter++
	}

	// Save ground truth JSON
	truthPath := filepath.Join(outDir, "ground_truth.json")
	truthBytes, _ := json.MarshalIndent(allTruths, "", "  ")
	if err := os.WriteFile(truthPath, truthBytes, 0644); err != nil {
		log.Fatalf("Failed to save ground truth: %v", err)
	}

	log.Printf("Successfully generated 30 bill images in %s and ground truth in %s", imgDir, truthPath)
}

func renderBillImage(destPath string, inv models.InvoiceExtraction, city, phone, anomaly string) {
	width := 800
	height := 1050

	bgCol := color.RGBA{R: 250, G: 250, B: 247, A: 255}
	if anomaly == "faded_thermal" {
		bgCol = color.RGBA{R: 235, G: 235, B: 230, A: 255}
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{bgCol}, image.Point{}, draw.Src)

	// Border
	drawRect(img, 20, 20, width-40, height-40, color.RGBA{180, 180, 180, 255})

	// Header Text
	textCol := color.RGBA{20, 20, 20, 255}
	if anomaly == "faded_thermal" {
		textCol = color.RGBA{140, 140, 135, 255} // Low contrast
	}

	y := 60
	vendorTitle := strings.ToUpper(inv.VendorName)
	if anomaly == "hindi_header" {
		vendorTitle = "शर्मा डिस्ट्रीब्यूटर्स (SHARMA DISTRIBUTORS)"
	}
	drawText(img, 40, y, "TAX INVOICE / बिल", textCol)
	y += 30
	drawText(img, 40, y, vendorTitle, textCol)
	y += 25
	drawText(img, 40, y, fmt.Sprintf("GSTIN: %s  |  City: %s  |  Ph: %s", inv.VendorGSTIN, city, phone), textCol)
	y += 35
	drawLine(img, 40, y, width-40, y, color.RGBA{200, 200, 200, 255})
	y += 25

	drawText(img, 40, y, fmt.Sprintf("Billed To: GUPTA GENERAL STORE (Kirana & FMCG)"), textCol)
	drawText(img, 500, y, fmt.Sprintf("Invoice #: %s", inv.InvoiceNumber), textCol)
	y += 25
	drawText(img, 40, y, fmt.Sprintf("Buyer GSTIN: 07AABCG1234D1Z8"), textCol)
	drawText(img, 500, y, fmt.Sprintf("Date: %s", inv.InvoiceDate), textCol)
	y += 35
	drawLine(img, 40, y, width-40, y, color.RGBA{200, 200, 200, 255})
	y += 30

	// Table Header
	drawText(img, 40, y, "Item Description", textCol)
	drawText(img, 360, y, "Pack", textCol)
	drawText(img, 440, y, "Qty", textCol)
	drawText(img, 520, y, "Rate", textCol)
	drawText(img, 620, y, "Taxable (Rs)", textCol)
	y += 20
	drawLine(img, 40, y, width-40, y, color.RGBA{160, 160, 160, 255})
	y += 25

	for _, it := range inv.LineItems {
		drawText(img, 40, y, it.Description, textCol)
		drawText(img, 360, y, it.PackSize, textCol)
		drawText(img, 440, y, fmt.Sprintf("%.0f %s", it.Quantity, it.Unit), textCol)
		drawText(img, 520, y, fmt.Sprintf("%.2f", it.UnitPrice), textCol)
		drawText(img, 620, y, fmt.Sprintf("%.2f", it.TaxableValue), textCol)
		y += 35
	}

	y += 30
	drawLine(img, 400, y, width-40, y, color.RGBA{200, 200, 200, 255})
	y += 25
	drawText(img, 440, y, "Subtotal:", textCol)
	drawText(img, 620, y, fmt.Sprintf("Rs %.2f", inv.Subtotal), textCol)
	y += 25
	drawText(img, 440, y, "CGST (Central Tax):", textCol)
	drawText(img, 620, y, fmt.Sprintf("Rs %.2f", inv.CGST), textCol)
	y += 25
	drawText(img, 440, y, "SGST (State Tax):", textCol)
	drawText(img, 620, y, fmt.Sprintf("Rs %.2f", inv.SGST), textCol)
	y += 25
	drawLine(img, 400, y, width-40, y, color.RGBA{100, 100, 100, 255})
	y += 30

	totalLabel := "GRAND TOTAL:"
	totalVal := fmt.Sprintf("Rs %.2f", inv.Total)
	if anomaly == "bad_math" {
		totalVal += " [Handwritten mismatch]"
	}
	drawText(img, 440, y, totalLabel, textCol)
	drawText(img, 620, y, totalVal, textCol)

	// Rubber stamp / signature placeholder
	y += 80
	drawRect(img, 500, y, 220, 80, color.RGBA{50, 80, 180, 180})
	drawText(img, 520, y+35, "AUTHORIZED SIGNATORY", color.RGBA{50, 80, 180, 255})
	drawText(img, 530, y+55, distTitle(inv.VendorName), color.RGBA{50, 80, 180, 255})

	// Rotate image if specified
	var finalImg image.Image = img
	if anomaly == "rotated_image" {
		finalImg = rotate90(img)
	}

	f, err := os.Create(destPath)
	if err != nil {
		log.Printf("Failed to create image: %v", err)
		return
	}
	defer f.Close()

	_ = jpeg.Encode(f, finalImg, &jpeg.Options{Quality: 88})
}

func distTitle(s string) string {
	if len(s) > 20 {
		return s[:20]
	}
	return s
}

func drawText(img *image.RGBA, x, y int, label string, col color.RGBA) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA) {
	for x := x1; x <= x2; x++ {
		img.Set(x, y1, col)
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	for i := x; i < x+w; i++ {
		img.Set(i, y, col)
		img.Set(i, y+h, col)
	}
	for j := y; j < y+h; j++ {
		img.Set(x, j, col)
		img.Set(x+w, j, col)
	}
}

func rotate90(src *image.RGBA) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			dst.Set(bounds.Dy()-y-1, x, src.At(x, y))
		}
	}
	return dst
}
