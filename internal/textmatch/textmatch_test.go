package textmatch

import "testing"

func TestTokens(t *testing.T) {
	got := Tokens("  Ecology   REPORT.pdf — ")
	if len(got) != 2 || got[0] != "ecology" || got[1] != "report.pdf" {
		t.Fatalf("Tokens = %v", got)
	}
	if len(Tokens("  ,. ")) != 0 {
		t.Fatal("punctuation-only query should produce no tokens")
	}
	if len(Tokens("")) != 0 {
		t.Fatal("empty query should produce no tokens")
	}
}

func TestDice(t *testing.T) {
	if got := Dice("ecology", "ecology"); got != 1 {
		t.Fatalf("identical strings = %v, want 1", got)
	}
	if got := Dice("ecology", "ecology"); got > 1.0001 {
		t.Fatalf("Dice above 1: %v", got)
	}
	if got := Dice("abc", "xyz"); got != 0 {
		t.Fatalf("unrelated strings = %v, want 0", got)
	}
	partial := Dice("ecolgy", "ecology")
	if partial < 0.4 || partial > 1 {
		t.Fatalf("typo similarity = %v, want within 0.4..1", partial)
	}
	if got := Dice("", "anything"); got != 0 {
		t.Fatalf("empty = %v, want 0", got)
	}
	if got := Dice("a", "a"); got != 1 {
		t.Fatalf("single rune equal = %v, want 1", got)
	}
}
