// Package textmatch holds the small string-similarity helpers shared by the
// catalog scorer and the rank engine (bigram Dice + tokenising). Keeping them
// in one place means /ask and Find agree on what "similar" means.
package textmatch

import (
	"strings"
	"unicode"
)

// Tokens splits a query into lowercased words, dropping punctuation-only
// fragments. "Ecology  report.pdf" → ["ecology", "report.pdf"].
func Tokens(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if !hasAlnum(f) {
			continue // punctuation-only fragment ("—", "()", "..")
		}
		out = append(out, f)
	}
	return out
}

// hasAlnum reports whether s contains a letter or a digit.
func hasAlnum(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// Dice is the Sørensen–Dice coefficient over character bigrams, a close,
// dependency-free stand-in for difflib.SequenceMatcher.ratio. Range 0..1.
func Dice(a, b string) float64 {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 || len(br) == 0 {
		return 0
	}
	if len(ar) == 1 && len(br) == 1 {
		if ar[0] == br[0] {
			return 1
		}
		return 0
	}
	counts := make(map[string]int, len(ar))
	for i := 0; i+1 < len(ar); i++ {
		counts[string(ar[i:i+2])]++
	}
	if len(counts) == 0 {
		if strings.Contains(b, a) || strings.Contains(a, b) {
			return 0.6
		}
		return 0
	}
	var inter int
	for i := 0; i+1 < len(br); i++ {
		bg := string(br[i : i+2])
		if counts[bg] > 0 {
			counts[bg]--
			inter++
		}
	}
	den := (len(ar) - 1) + (len(br) - 1)
	if den == 0 {
		return 0
	}
	return 2 * float64(inter) / float64(den)
}
