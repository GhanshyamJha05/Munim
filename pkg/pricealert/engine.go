package pricealert

import (
	"fmt"
	"sort"
	"time"

	"github.com/munim-app/munim/pkg/matcher"
	"github.com/munim-app/munim/pkg/models"
)

// HistoricalPurchase captures a previous purchase of a product for comparison.
type HistoricalPurchase struct {
	BillID        int64
	InvoiceNumber string
	ImagePath     string
	BillDate      time.Time
	SupplierName  string
	ItemDesc      string
	PackSize      string
	HSNCode       string
	UnitPrice     float64
	Quantity      float64
}

// Config controls price alert trigger thresholds.
type Config struct {
	SpikeThresholdPercent float64 // default: 3.0 (+3%)
	MaxDaysCheaperElse    int     // default: 45 days
}

// DefaultConfig provides recommended settings (+3% spike threshold).
func DefaultConfig() Config {
	return Config{
		SpikeThresholdPercent: 3.0,
		MaxDaysCheaperElse:     45,
	}
}

// Engine evaluates newly extracted items against past purchase history.
type Engine struct {
	cfg Config
}

func NewEngine(cfg Config) *Engine {
	if cfg.SpikeThresholdPercent <= 0 {
		cfg.SpikeThresholdPercent = 3.0
	}
	if cfg.MaxDaysCheaperElse <= 0 {
		cfg.MaxDaysCheaperElse = 45
	}
	return &Engine{cfg: cfg}
}

// CheckItemPrice evaluates a newly purchased item against historical purchases.
// Requirement 3: EXACTLY ONE alert per (supplier, item, bill).
// Shows invoice numbers (INV-...) as evidence, linking to each bill image, never internal IDs.
func (e *Engine) CheckItemPrice(
	currentBillID int64,
	currentInvoiceNumber string,
	currentImagePath string,
	currentBillDate time.Time,
	supplier string,
	item models.LineItem,
	history []HistoricalPurchase,
) []models.PriceAlert {
	var alerts []models.PriceAlert
	if item.UnitPrice <= 0 {
		return alerts
	}

	normCurrent := matcher.NormalizeItem(item.Description, item.PackSize, item.HSNCode)

	// Filter history to strictly matching items
	var sameSupplierHistory []HistoricalPurchase
	var otherSupplierHistory []HistoricalPurchase

	for _, h := range history {
		// Ignore current bill itself or items after current bill date
		if h.BillID == currentBillID || h.BillDate.After(currentBillDate) {
			continue
		}

		normH := matcher.NormalizeItem(h.ItemDesc, h.PackSize, h.HSNCode)
		if matched, _ := matcher.AreItemsMatch(normCurrent, normH); matched {
			if h.SupplierName == supplier {
				sameSupplierHistory = append(sameSupplierHistory, h)
			} else {
				otherSupplierHistory = append(otherSupplierHistory, h)
			}
		}
	}

	// Sort same supplier history descending by date (most recent first)
	sort.Slice(sameSupplierHistory, func(i, j int) bool {
		return sameSupplierHistory[i].BillDate.After(sameSupplierHistory[j].BillDate)
	})

	// Check same-supplier price hike:
	// A) Last bill comparison
	var prevBill *HistoricalPurchase
	var pctChangePrior float64
	var priorHike bool

	if len(sameSupplierHistory) > 0 {
		prev := sameSupplierHistory[0]
		pctChangePrior = ((item.UnitPrice - prev.UnitPrice) / prev.UnitPrice) * 100.0
		if pctChangePrior >= e.cfg.SpikeThresholdPercent {
			priorHike = true
			prevBill = &prev
		}
	}

	// B) Trailing 3-bill median comparison
	var medianRefBill *HistoricalPurchase
	var pctChangeMedian float64
	var medianPrice float64
	var medianHike bool

	if len(sameSupplierHistory) >= 3 {
		recentThree := sameSupplierHistory[:3]
		prices := make([]float64, 3)
		for i, p := range recentThree {
			prices[i] = p.UnitPrice
		}
		sort.Float64s(prices)
		medianPrice = prices[1] // middle element of 3

		pctChangeMedian = ((item.UnitPrice - medianPrice) / medianPrice) * 100.0
		if pctChangeMedian >= e.cfg.SpikeThresholdPercent {
			medianHike = true
			for _, r := range recentThree {
				if r.UnitPrice == medianPrice {
					medianCopy := r
					medianRefBill = &medianCopy
					break
				}
			}
		}
	}

	// CONSOLIDATION: If hike exists, emit EXACTLY ONE alert for this (supplier, item, bill)
	if priorHike && medianHike && prevBill != nil && medianRefBill != nil {
		// Both comparisons as evidence in a single alert!
		msg := fmt.Sprintf("%s is %.1f%% higher than your last bill from %s (₹%.2f -> ₹%.2f) and %.1f%% above 3-bill median (₹%.2f).",
			item.Description, pctChangePrior, supplier, prevBill.UnitPrice, item.UnitPrice, pctChangeMedian, medianPrice)

		alerts = append(alerts, models.PriceAlert{
			Type:                   "spike_both",
			ItemDescription:        item.Description,
			ItemNormalized:         normCurrent.CleanName,
			PackSize:               normCurrent.PackSize,
			CurrentBillID:          currentBillID,
			CurrentInvoiceNumber:   currentInvoiceNumber,
			CurrentImagePath:       currentImagePath,
			CurrentBillDate:        currentBillDate,
			CurrentSupplier:        supplier,
			CurrentPrice:           item.UnitPrice,
			ReferenceBillID:        prevBill.BillID,
			ReferenceInvoiceNumber: prevBill.InvoiceNumber,
			ReferenceImagePath:     prevBill.ImagePath,
			ReferenceBillDate:      prevBill.BillDate,
			ReferenceSupplier:      supplier,
			ReferencePrice:         prevBill.UnitPrice,
			PercentageDiff:         pctChangePrior,
			Message:                msg,
			CreatedAt:              time.Now(),
		})
	} else if priorHike && prevBill != nil {
		msg := fmt.Sprintf("%s is %.1f%% higher than on your last bill from %s (₹%.2f -> ₹%.2f).",
			item.Description, pctChangePrior, supplier, prevBill.UnitPrice, item.UnitPrice)

		alerts = append(alerts, models.PriceAlert{
			Type:                   "spike_prior",
			ItemDescription:        item.Description,
			ItemNormalized:         normCurrent.CleanName,
			PackSize:               normCurrent.PackSize,
			CurrentBillID:          currentBillID,
			CurrentInvoiceNumber:   currentInvoiceNumber,
			CurrentImagePath:       currentImagePath,
			CurrentBillDate:        currentBillDate,
			CurrentSupplier:        supplier,
			CurrentPrice:           item.UnitPrice,
			ReferenceBillID:        prevBill.BillID,
			ReferenceInvoiceNumber: prevBill.InvoiceNumber,
			ReferenceImagePath:     prevBill.ImagePath,
			ReferenceBillDate:      prevBill.BillDate,
			ReferenceSupplier:      supplier,
			ReferencePrice:         prevBill.UnitPrice,
			PercentageDiff:         pctChangePrior,
			Message:                msg,
			CreatedAt:              time.Now(),
		})
	} else if medianHike && medianRefBill != nil {
		msg := fmt.Sprintf("%s is %.1f%% higher than 3-bill median from %s (Median: ₹%.2f -> ₹%.2f).",
			item.Description, pctChangeMedian, supplier, medianPrice, item.UnitPrice)

		alerts = append(alerts, models.PriceAlert{
			Type:                   "spike_median",
			ItemDescription:        item.Description,
			ItemNormalized:         normCurrent.CleanName,
			PackSize:               normCurrent.PackSize,
			CurrentBillID:          currentBillID,
			CurrentInvoiceNumber:   currentInvoiceNumber,
			CurrentImagePath:       currentImagePath,
			CurrentBillDate:        currentBillDate,
			CurrentSupplier:        supplier,
			CurrentPrice:           item.UnitPrice,
			ReferenceBillID:        medianRefBill.BillID,
			ReferenceInvoiceNumber: medianRefBill.InvoiceNumber,
			ReferenceImagePath:     medianRefBill.ImagePath,
			ReferenceBillDate:      medianRefBill.BillDate,
			ReferenceSupplier:      supplier,
			ReferencePrice:         medianPrice,
			PercentageDiff:         pctChangeMedian,
			Message:                msg,
			CreatedAt:              time.Now(),
		})
	} else {
		// Only check cheaper elsewhere if no same-supplier hike was emitted
		cutoff := currentBillDate.AddDate(0, 0, -e.cfg.MaxDaysCheaperElse)
		var bestAlternative *HistoricalPurchase

		for _, alt := range otherSupplierHistory {
			if alt.BillDate.Before(cutoff) {
				continue
			}
			if alt.UnitPrice < item.UnitPrice {
				if bestAlternative == nil || alt.UnitPrice < bestAlternative.UnitPrice {
					bestCopy := alt
					bestAlternative = &bestCopy
				}
			}
		}

		if bestAlternative != nil {
			savingsPct := ((item.UnitPrice - bestAlternative.UnitPrice) / item.UnitPrice) * 100.0
			if savingsPct >= e.cfg.SpikeThresholdPercent {
				msg := fmt.Sprintf("%s is available for ₹%.2f from %s (save %.1f%% vs ₹%.2f from %s).",
					item.Description, bestAlternative.UnitPrice, bestAlternative.SupplierName, savingsPct, item.UnitPrice, supplier)

				alerts = append(alerts, models.PriceAlert{
					Type:                   "cheaper_elsewhere",
					ItemDescription:        item.Description,
					ItemNormalized:         normCurrent.CleanName,
					PackSize:               normCurrent.PackSize,
					CurrentBillID:          currentBillID,
					CurrentInvoiceNumber:   currentInvoiceNumber,
					CurrentImagePath:       currentImagePath,
					CurrentBillDate:        currentBillDate,
					CurrentSupplier:        supplier,
					CurrentPrice:           item.UnitPrice,
					ReferenceBillID:        bestAlternative.BillID,
					ReferenceInvoiceNumber: bestAlternative.InvoiceNumber,
					ReferenceImagePath:     bestAlternative.ImagePath,
					ReferenceBillDate:      bestAlternative.BillDate,
					ReferenceSupplier:      bestAlternative.SupplierName,
					ReferencePrice:         bestAlternative.UnitPrice,
					PercentageDiff:         -savingsPct,
					Message:                msg,
					CreatedAt:              time.Now(),
				})
			}
		}
	}

	return alerts
}
