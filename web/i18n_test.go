package web_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestAllUIStringsHaveHindiTranslation ensures every key in translations.en exists and is non-empty in translations.hi
func TestAllUIStringsHaveHindiTranslation(t *testing.T) {
	appJS, err := os.ReadFile("app.js")
	if err != nil {
		t.Fatalf("Failed to read app.js: %v", err)
	}

	content := string(appJS)

	// Extract the translations object using regex or substring
	startIdx := strings.Index(content, "const translations = {")
	if startIdx == -1 {
		t.Fatalf("Could not find 'const translations = {' in app.js")
	}
	endIdx := strings.Index(content[startIdx:], "};")
	if endIdx == -1 {
		t.Fatalf("Could not find closing '};' for translations object")
	}
	transBlock := content[startIdx : startIdx+endIdx+2]

	// Extract en block
	enStart := strings.Index(transBlock, "en: {")
	enEnd := strings.Index(transBlock[enStart:], "},")
	if enStart == -1 || enEnd == -1 {
		t.Fatalf("Could not locate en translations block")
	}
	enContent := transBlock[enStart+5 : enStart+enEnd]

	// Extract hi block
	hiStart := strings.Index(transBlock, "hi: {")
	hiEnd := strings.Index(transBlock[hiStart:], "}")
	if hiStart == -1 || hiEnd == -1 {
		t.Fatalf("Could not locate hi translations block")
	}
	hiContent := transBlock[hiStart+5 : hiStart+hiEnd]

	// Parse keys and values from JS object lines: key: "value",
	keyValRegex := regexp.MustCompile(`(?m)^\s*([a-zA-Z0-9_]+)\s*:\s*"(.*)"\s*,?`)

	enMatches := keyValRegex.FindAllStringSubmatch(enContent, -1)
	hiMatches := keyValRegex.FindAllStringSubmatch(hiContent, -1)

	if len(enMatches) == 0 {
		t.Fatalf("No english translation keys extracted")
	}

	hiMap := make(map[string]string)
	for _, m := range hiMatches {
		hiMap[m[1]] = m[2]
	}

	var missingKeys []string
	var emptyKeys []string

	for _, m := range enMatches {
		key := m[1]
		hiVal, exists := hiMap[key]
		if !exists {
			missingKeys = append(missingKeys, key)
		} else if strings.TrimSpace(hiVal) == "" {
			emptyKeys = append(emptyKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		t.Errorf("Missing Hindi translations for %d keys: %v", len(missingKeys), missingKeys)
	}
	if len(emptyKeys) > 0 {
		t.Errorf("Empty Hindi translations for %d keys: %v", len(emptyKeys), emptyKeys)
	}

	t.Logf("Successfully verified %d translation keys: 100%% complete Hindi coverage", len(enMatches))
}
