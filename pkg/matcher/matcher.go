package matcher

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

// PackSizeRegex extracts numeric capacity + unit (e.g. "100g", "1 kg", "500 ml", "12-pack", "1.5 ltr")
var (
	packSizePattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(kg|kilo|kilogram|g|gm|gms|gram|grams|ml|milli|ltr|l|litre|litres|liter|liters|pcs|pc|pack|packets|pkt|units)`)
	packCountPattern = regexp.MustCompile(`(?i)(\d+)\s*(?:-|x|\*|\s)?(?:pack|pkt|pcs|pc)`)
	cleanPunctuation = regexp.MustCompile(`[^\w\s\.\-]`)
)

// NormalizedItem stores clean representation of a product name and its exact pack size.
type NormalizedItem struct {
	Original    string
	CleanName   string
	PackSize    string // Standardized e.g. "100g", "1000g", "12pack", "1000ml"
	HSNCode     string
}

// NormalizeItem processes item text, unifies units, and extracts explicit pack sizes.
func NormalizeItem(rawDescription string, rawPackSize string, hsn string) NormalizedItem {
	clean := strings.ToLower(rawDescription)
	clean = cleanPunctuation.ReplaceAllString(clean, " ")
	clean = strings.Join(strings.Fields(clean), " ")

	extractedPack := ""

	// Check explicit pack size first if provided
	if rawPackSize != "" {
		extractedPack = StandardizePackSize(rawPackSize)
	}

	// If pack size wasn't in explicit field or was generic, look in description
	if extractedPack == "" {
		if match := packCountPattern.FindStringSubmatch(clean); len(match) > 1 {
			extractedPack = match[1] + "pack"
		} else if match := packSizePattern.FindStringSubmatch(clean); len(match) > 2 {
			extractedPack = StandardizePackSize(match[1] + match[2])
		}
	}

	// Remove pack size text from clean name to isolate brand & product core
	nameWithoutPack := clean
	if extractedPack != "" {
		nameWithoutPack = packSizePattern.ReplaceAllString(nameWithoutPack, "")
		nameWithoutPack = packCountPattern.ReplaceAllString(nameWithoutPack, "")
		nameWithoutPack = strings.Join(strings.Fields(nameWithoutPack), " ")
	}

	return NormalizedItem{
		Original:  rawDescription,
		CleanName: nameWithoutPack,
		PackSize:  extractedPack,
		HSNCode:   strings.TrimSpace(hsn),
	}
}

// StandardizePackSize converts variations of units to standard metric grams/millilitres/units.
func StandardizePackSize(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if m := packCountPattern.FindStringSubmatch(raw); len(m) > 1 {
		return m[1] + "pack"
	}
	m := packSizePattern.FindStringSubmatch(raw)
	if len(m) < 3 {
		return raw
	}

	valStr := strings.TrimSpace(m[1])
	unit := strings.TrimSpace(m[2])

	switch unit {
	case "kg", "kilo", "kilogram":
		return valStr + "kg"
	case "g", "gm", "gms", "gram", "grams":
		return valStr + "g"
	case "ltr", "l", "litre", "litres", "liter", "liters":
		return valStr + "l"
	case "ml", "milli":
		return valStr + "ml"
	case "pcs", "pc", "pack", "pkt", "packets", "units":
		return valStr + "pack"
	default:
		return valStr + unit
	}
}

var criticalDifferentiators = []string{
	"masala", "atta", "classic", "magic", "original", "gold", "cream",
	"chocolate", "vanilla", "butter", "salted", "sugar", "diet", "lite",
	"red", "blue", "green", "almond", "kaju", "pista", "cardamom", "elaichi",
}

// AreItemsMatch determines if two items represent the exact same product and pack.
// CRITICAL RULE: If pack sizes are specified and do not match (e.g. 100g vs 200g), match MUST be false.
func AreItemsMatch(a, b NormalizedItem) (bool, float64) {
	// 1. Pack size trap check: If both have recognized pack sizes and they differ, strictly false!
	if a.PackSize != "" && b.PackSize != "" && a.PackSize != b.PackSize {
		return false, 0.0
	}

	// 2. Flavour / Variant differentiator check: If one item has a differentiator and the other doesn't, reject!
	for _, diff := range criticalDifferentiators {
		hasA := containsWord(a.CleanName, diff)
		hasB := containsWord(b.CleanName, diff)
		if hasA != hasB {
			return false, 0.0
		}
	}

	// 3. Similarity on clean product names
	sim := JaroWinkler(a.CleanName, b.CleanName)

	// 4. HSN code tiebreaker bonus
	if a.HSNCode != "" && b.HSNCode != "" && a.HSNCode == b.HSNCode {
		sim += 0.10
		if sim > 1.0 {
			sim = 1.0
		}
	}

	// High confidence threshold for matching
	threshold := 0.82
	if sim >= threshold {
		return true, sim
	}

	return false, sim
}

func containsWord(s, word string) bool {
	fields := strings.Fields(s)
	for _, f := range fields {
		if f == word {
			return true
		}
	}
	return false
}

// JaroWinkler calculates the Jaro-Winkler string similarity distance between two strings (0.0 to 1.0).
func JaroWinkler(s1, s2 string) float64 {
	jaro := jaroDistance(s1, s2)
	if jaro < 0.7 {
		return jaro
	}

	// Calculate common prefix up to 4 characters
	prefixLen := 0
	maxPrefix := 4
	r1, r2 := []rune(s1), []rune(s2)
	minLen := len(r1)
	if len(r2) < minLen {
		minLen = len(r2)
	}
	if maxPrefix > minLen {
		maxPrefix = minLen
	}

	for i := 0; i < maxPrefix; i++ {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	p := 0.1 // scaling factor
	return jaro + float64(prefixLen)*p*(1.0-jaro)
}

func jaroDistance(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	l1, l2 := len(r1), len(r2)

	if l1 == 0 && l2 == 0 {
		return 1.0
	}
	if l1 == 0 || l2 == 0 {
		return 0.0
	}

	matchDistance := int(math.Floor(math.Max(float64(l1), float64(l2))/2.0)) - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, l1)
	s2Matches := make([]bool, l2)

	matches := 0
	for i := 0; i < l1; i++ {
		start := int(math.Max(0, float64(i-matchDistance)))
		end := int(math.Min(float64(i+matchDistance+1), float64(l2)))

		for j := start; j < end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	transpositions := 0
	k := 0
	for i := 0; i < l1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(l1) + m/float64(l2) + (m-float64(transpositions)/2.0)/m) / 3.0
}

// CleanWhitespace trims extra whitespace
func CleanWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// IsASCII reports whether string is pure ASCII or contains Hindi/regional script
func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
