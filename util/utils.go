package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SplitPriceAndCurrency splits the price and currency
func SplitPriceAndCurrency(input string) (float64, string, error) {
	re := regexp.MustCompile(`^([\d,]+)\s+([A-Z]{2})$`)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 3 {
		priceStr := matches[1]
		currency := matches[2]

		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return price, "", fmt.Errorf("invalid input format")
		}

		return price, currency, nil
	}

	return 0.0, "", fmt.Errorf("invalid input format")
}

// FilterXMLs filters a list of collected XMLs based on a pattern.
func FilterXMLs(collectedXMLs []string, pattern string) []string {
	var filtered []string
	for _, xml := range collectedXMLs {
		if strings.Contains(xml, pattern) {
			filtered = append(filtered, xml)
		}
	}
	return filtered
}
