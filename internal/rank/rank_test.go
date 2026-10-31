package rank

import (
	"testing"
	"time"
)

func cand(name, dir string, mtime time.Time, size int64) Candidate {
	return Candidate{
		Path:  dir + "/" + name,
		Name:  name,
		Dir:   dir,
		Ext:   extOf(name),
		Size:  size,
		MTime: mtime.Unix(),
	}
}

func extOf(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}

func TestExactNameRule(t *testing.T) {
	r := exactNameRule{}
	q := NewQuery("ecology report")
	if s, why := r.Score(q, cand("Ecology Report.pdf", "/x", time.Now(), 1)); s != 1 || why != "exact name" {
		t.Fatalf("exact name (extension ignored) = %v %q, want 1 exact name", s, why)
	}
	if s, _ := r.Score(NewQuery("ecology report.pdf"), cand("ecology report.pdf", "/x", time.Now(), 1)); s != 1 {
		t.Fatalf("exact name with extension = %v, want 1", s)
	}
	if s, _ := r.Score(q, cand("ecology-report-notes.txt", "/x", time.Now(), 1)); s != 0 {
		t.Fatalf("non-exact name = %v, want 0", s)
	}
}

func TestNameContainsFraction(t *testing.T) {
	r := nameContainsRule{}
	q := NewQuery("ecology report 2026")
	// two of three tokens present -> 2/3
	if s, _ := r.Score(q, cand("ecology-report.pdf", "/x", time.Now(), 1)); s < 0.66 || s > 0.67 {
		t.Fatalf("partial token match = %v, want ~0.667", s)
	}
	if s, _ := r.Score(NewQuery("budget"), cand("ecology.pdf", "/x", time.Now(), 1)); s != 0 {
		t.Fatalf("no token match = %v, want 0", s)
	}
	if s, _ := r.Score(NewQuery("ECOLOGY"), cand("Ecology_2026.pdf", "/x", time.Now(), 1)); s != 1 {
		t.Fatalf("case-insensitive contains = %v, want 1", s)
	}
}

func TestExtensionRule(t *testing.T) {
	r := extensionRule{}
	if s, why := r.Score(NewQuery("report pdf"), cand("report.pdf", "/x", time.Now(), 1)); s != 1 || why != "extension" {
		t.Fatalf("extension hint = %v %q", s, why)
	}
	if s, _ := r.Score(NewQuery("*.docx"), cand("letter.docx", "/x", time.Now(), 1)); s != 1 {
		t.Fatalf("glob-style extension = %v, want 1", s)
	}
	if s, _ := r.Score(NewQuery("report pdf"), cand("report.txt", "/x", time.Now(), 1)); s != 0 {
		t.Fatalf("wrong extension = %v, want 0", s)
	}
	if s, _ := r.Score(NewQuery("readme pdf"), cand("README", "/x", time.Now(), 1)); s != 0 {
		t.Fatalf("no extension on candidate = %v, want 0", s)
	}
}

func TestRecentlyModifiedDecay(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	r := recentlyModifiedRule{now: func() time.Time { return now }}
	fresh, _ := r.Score(NewQuery("x"), cand("x.txt", "/x", now, 1))
	if fresh < 0.99 {
		t.Fatalf("fresh file score = %v, want ~1", fresh)
	}
	half, _ := r.Score(NewQuery("x"), cand("x.txt", "/x", now.AddDate(0, 0, -30), 1))
	if half < 0.49 || half > 0.51 {
		t.Fatalf("30-day half-life score = %v, want ~0.5", half)
	}
	old, _ := r.Score(NewQuery("x"), cand("x.txt", "/x", now.AddDate(0, 0, -365), 1))
	if old > 0.01 {
		t.Fatalf("year-old file score = %v, want ~0", old)
	}
	unknown, _ := r.Score(NewQuery("x"), Candidate{Name: "x.txt"})
	if unknown != 0 {
		t.Fatalf("unknown mtime = %v, want 0", unknown)
	}
}

func TestFolderHintRule(t *testing.T) {
	r := folderHintRule{}
	c := cand("notes.txt", "/archive/reports/ecology", time.Now(), 1)
	if s, why := r.Score(NewQuery("ecology"), c); s != 1 || why != "same folder" {
		t.Fatalf("folder hint = %v %q", s, why)
	}
	if s, _ := r.Score(NewQuery("budget"), c); s != 0 {
		t.Fatalf("unrelated folder = %v, want 0", s)
	}
}

func TestFuzzyNameRule(t *testing.T) {
	r := fuzzyNameRule{}
	close, _ := r.Score(NewQuery("ecolgy"), cand("ecology.pdf", "/x", time.Now(), 1))
	if close <= 0.4 {
		t.Fatalf("typo score = %v, want > 0.4", close)
	}
	none, _ := r.Score(NewQuery("zzzz"), cand("ecology.pdf", "/x", time.Now(), 1))
	if none != 0 {
		t.Fatalf("unrelated name = %v, want 0", none)
	}
}

func TestRulesSelfRegister(t *testing.T) {
	want := []string{"exact_name", "name_contains", "extension", "recently_modified", "folder_hint", "fuzzy_name"}
	got := Registered()
	if len(got) != len(want) {
		t.Fatalf("registered %d rules, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID() != id {
			t.Errorf("rule %d = %q, want %q", i, got[i].ID(), id)
		}
		if got[i].Label() == "" {
			t.Errorf("rule %q has no label", id)
		}
	}
}

func TestDefaultConfigMatchesSpec(t *testing.T) {
	want := map[string]int{
		"exact_name": 100, "name_contains": 78, "extension": 55,
		"recently_modified": 42, "folder_hint": 28, "fuzzy_name": 36,
	}
	cfg := DefaultConfig()
	if len(cfg) != len(want) {
		t.Fatalf("default config has %d rules, want %d", len(cfg), len(want))
	}
	for _, c := range cfg {
		if w, ok := want[c.ID]; !ok || w != c.Weight {
			t.Errorf("default weight for %q = %d, want %d", c.ID, c.Weight, want[c.ID])
		}
		if !c.Enabled {
			t.Errorf("rule %q should be enabled by default", c.ID)
		}
	}
}

func TestWeightsChangeOrder(t *testing.T) {
	now := time.Now()
	q := NewQuery("ecology")
	exact := cand("ecology.pdf", "/a", now, 1)
	contains := cand("ecology-report.docx", "/b", now, 1)

	// Default weights favour the exact name.
	e := NewEngine(DefaultConfig())
	hits := e.Rank(q, []Candidate{contains, exact}, 0.01, 10)
	if len(hits) != 2 {
		t.Fatalf("hits = %d, want 2", len(hits))
	}
	if hits[0].Name != "ecology.pdf" {
		t.Fatalf("top hit = %q, want ecology.pdf", hits[0].Name)
	}

	// Zero the exact-name rule: a candidate matching every token in its name
	// now beats the one that only shares one token.
	cfg := DefaultConfig()
	for i := range cfg {
		if cfg[i].ID == "exact_name" {
			cfg[i].Weight = 0
		}
	}
	partial := cand("ecology.pdf", "/a", now, 1)
	full := cand("ecology report notes.docx", "/b", now, 1)
	q2 := NewQuery("ecology report")
	hits2 := NewEngine(cfg).Rank(q2, []Candidate{partial, full}, 0.01, 10)
	if len(hits2) == 0 || hits2[0].Name != "ecology report notes.docx" {
		t.Fatalf("with exact disabled top hit = %v", hits2)
	}

	// Disabling a rule removes its contribution entirely.
	cfg3 := DefaultConfig()
	for i := range cfg3 {
		cfg3[i].Enabled = false
	}
	if hits3 := NewEngine(cfg3).Rank(q, []Candidate{exact}, 0.01, 10); len(hits3) != 0 {
		t.Fatalf("all rules disabled still produced %v", hits3)
	}
}

func TestTieBreakUsesConfigOrder(t *testing.T) {
	now := time.Now()
	// Two files scoring identically on every rule; only the name differs, so
	// build a query where both are "contains" hits of equal fraction.
	a := cand("notes-a.txt", "/same/dir", now, 10)
	b := cand("notes-b.txt", "/same/dir", now, 10)
	q := NewQuery("notes")

	cfg := DefaultConfig()
	for i := range cfg {
		cfg[i].Enabled = cfg[i].ID == "name_contains" || cfg[i].ID == "folder_hint"
	}
	first := NewEngine(cfg).Rank(q, []Candidate{b, a}, 0.01, 10)
	if len(first) != 2 {
		t.Fatalf("hits = %d", len(first))
	}
	// Deterministic fallback: name ascending when the vectors tie.
	if first[0].Name != "notes-a.txt" {
		t.Fatalf("tie order = %q, want notes-a.txt", first[0].Name)
	}
}

func TestReasonIsStrongestRule(t *testing.T) {
	now := time.Now()
	e := NewEngine(DefaultConfig())
	hits := e.Rank(NewQuery("ecology"), []Candidate{
		cand("ecology.pdf", "/x", now, 1),
		cand("ecology-notes.txt", "/x", now, 1),
	}, 0.01, 10)
	if hits[0].Reason != "exact name" {
		t.Fatalf("reason = %q, want exact name", hits[0].Reason)
	}
}

func TestCutoffDropsWeakHits(t *testing.T) {
	now := time.Now()
	e := NewEngine(DefaultConfig())
	// "zzz" only weakly matches the folder/fuzzy rules.
	hits := e.Rank(NewQuery("zzz"), []Candidate{cand("ecology.pdf", "/x", now, 1)}, DefaultCutoff, 10)
	if len(hits) != 0 {
		t.Fatalf("weak match survived the cutoff: %v", hits)
	}
}

func TestScoreIsTwoDecimals(t *testing.T) {
	now := time.Now()
	e := NewEngine(DefaultConfig())
	hits := e.Rank(NewQuery("ecology"), []Candidate{cand("ecology.pdf", "/x", now, 1)}, 0.01, 10)
	if len(hits) != 1 {
		t.Fatalf("hits = %d", len(hits))
	}
	scaled := hits[0].Score * 100
	if scaled != float64(int64(scaled)) {
		t.Fatalf("score %v is not rounded to 2 decimals", hits[0].Score)
	}
}

func TestParseConfigFallsBackAndNormalises(t *testing.T) {
	if got := ParseConfig("not json"); len(got) != len(DefaultConfig()) {
		t.Fatalf("corrupt JSON should fall back to defaults, got %v", got)
	}
	if got := ParseConfig(""); len(got) != len(DefaultConfig()) {
		t.Fatalf("empty JSON should fall back to defaults, got %v", got)
	}
	raw := `[{"id":"exact_name","enabled":false,"weight":500},{"id":"ghost","enabled":true,"weight":10}]`
	got := ParseConfig(raw)
	if got[0].ID != "exact_name" || got[0].Enabled || got[0].Weight != 100 {
		t.Fatalf("clamp/disable failed: %+v", got[0])
	}
	for _, c := range got {
		if c.ID == "ghost" {
			t.Fatal("unknown rule survived normalisation")
		}
	}
	// Missing rules are appended so new criteria apply automatically.
	if len(got) != len(DefaultConfig()) {
		t.Fatalf("normalised config has %d rules, want %d", len(got), len(DefaultConfig()))
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg[1].Weight = 13
	cfg[2].Enabled = false
	raw, err := MarshalConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	back := ParseConfig(raw)
	if back[1].Weight != 13 || back[2].Enabled {
		t.Fatalf("round trip lost config: %+v", back)
	}
}
