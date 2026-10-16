package matcher

import (
	"fmt"
	"testing"
)

func TestItemNormalization(t *testing.T) {
	tests := []struct {
		name         string
		desc         string
		rawPack      string
		wantPack     string
		wantCleanSub string
	}{
		{
			name:         "Parle-G 12-pack",
			desc:         "Parle-G Gold 12-pack Biscuit",
			wantPack:     "12pack",
			wantCleanSub: "parle-g gold biscuit",
		},
		{
			name:         "Tata Salt 1kg",
			desc:         "Tata Salt Vacuum Evaporated (1 kg)",
			wantPack:     "1kg",
			wantCleanSub: "tata salt vacuum evaporated",
		},
		{
			name:         "Fortune Sunflower 1 Ltr",
			desc:         "Fortune Sunlite Refined Sunflower Oil 1 Ltr Pouch",
			wantPack:     "1l",
			wantCleanSub: "fortune sunlite refined sunflower oil pouch",
		},
		{
			name:         "Dettol Soap 75g",
			desc:         "Dettol Original Soap 75 gm",
			wantPack:     "75g",
			wantCleanSub: "dettol original soap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			norm := NormalizeItem(tt.desc, tt.rawPack, "")
			if norm.PackSize != tt.wantPack {
				t.Errorf("NormalizeItem(%q).PackSize = %q, want %q", tt.desc, norm.PackSize, tt.wantPack)
			}
			fmt.Printf("Normalized: Clean=%q Pack=%q\n", norm.CleanName, norm.PackSize)
		})
	}
}

func TestAreItemsMatch(t *testing.T) {
	tests := []struct {
		name      string
		itemA     string
		packA     string
		hsnA      string
		itemB     string
		packB     string
		hsnB      string
		wantMatch bool
		reason    string
	}{
		{
			name:      "Exact Match with minor spelling/format differences",
			itemA:     "Parle-G 12-pack Biscuits",
			packA:     "12-pack",
			itemB:     "Parle G Biscuit 12 pack",
			packB:     "12 pack",
			wantMatch: true,
			reason:    "Standard kirana name variation",
		},
		{
			name:      "Same Brand Different Pack Sizes - MUST NOT MATCH",
			itemA:     "Tata Salt 1kg",
			packA:     "1kg",
			itemB:     "Tata Salt 500g",
			packB:     "500g",
			wantMatch: false,
			reason:    "Pack size trap: 1kg vs 500g",
		},
		{
			name:      "Different Flavor Same Brand - MUST NOT MATCH",
			itemA:     "Maggi 2-Minute Noodles Masala 70g",
			packA:     "70g",
			itemB:     "Maggi 2-Minute Noodles Atta 70g",
			packB:     "70g",
			wantMatch: false,
			reason:    "Flavor difference: Masala vs Atta",
		},
		{
			name:      "HSN Tiebreaker match",
			itemA:     "Aashirvaad Shudh Chakki Atta 10kg",
			packA:     "10kg",
			hsnA:      "1101",
			itemB:     "Ashirwad Chakki Atta 10 kg",
			packB:     "10kg",
			hsnB:      "1101",
			wantMatch: true,
			reason:    "Fuzzy phonetic match bolstered by HSN 1101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normA := NormalizeItem(tt.itemA, tt.packA, tt.hsnA)
			normB := NormalizeItem(tt.itemB, tt.packB, tt.hsnB)

			matched, score := AreItemsMatch(normA, normB)
			if matched != tt.wantMatch {
				t.Errorf("AreItemsMatch() = %v (score %.3f), want %v. Reason: %s", matched, score, tt.wantMatch, tt.reason)
			}
		})
	}
}
