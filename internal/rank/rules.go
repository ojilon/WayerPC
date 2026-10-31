package rank

import (
	"math"
	"path/filepath"
	"strings"
	"time"

	"wayerpc/internal/textmatch"
)

// The built-in rules. Each one is registered from init() so the engine, the
// settings JSON and the UI rule list all stay in sync automatically.

func init() {
	Register(exactNameRule{})
	Register(nameContainsRule{})
	Register(extensionRule{})
	Register(recentlyModifiedRule{})
	Register(folderHintRule{})
	Register(fuzzyNameRule{})
}

// exactNameRule: 1.0 when the query equals the filename without extension.
type exactNameRule struct{}

func (exactNameRule) ID() string    { return "exact_name" }
func (exactNameRule) Label() string { return "Exact name" }

func (exactNameRule) Score(q Query, c Candidate) (float64, string) {
	if q.Raw == "" {
		return 0, ""
	}
	if stripExt(c.Name) == strings.ToLower(strings.TrimSpace(q.Raw)) {
		return 1, "exact name"
	}
	// "report.pdf" typed in full also counts as an exact name.
	if strings.ToLower(c.Name) == strings.ToLower(strings.TrimSpace(q.Raw)) {
		return 1, "exact name"
	}
	return 0, ""
}

// nameContainsRule: fraction of query tokens present in the filename.
type nameContainsRule struct{}

func (nameContainsRule) ID() string    { return "name_contains" }
func (nameContainsRule) Label() string { return "Name contains" }

func (nameContainsRule) Score(q Query, c Candidate) (float64, string) {
	if len(q.Tokens) == 0 {
		return 0, ""
	}
	name := strings.ToLower(c.Name)
	found := 0
	for _, tok := range q.Tokens {
		if strings.Contains(name, tok) {
			found++
		}
	}
	if found == 0 {
		return 0, ""
	}
	return float64(found) / float64(len(q.Tokens)), "contains"
}

// extensionRule: the query names the file type ("pdf", "*.docx").
type extensionRule struct{}

func (extensionRule) ID() string    { return "extension" }
func (extensionRule) Label() string { return "Extension" }

func (extensionRule) Score(q Query, c Candidate) (float64, string) {
	ext := strings.ToLower(strings.TrimPrefix(c.Ext, "."))
	if ext == "" {
		return 0, ""
	}
	if q.ExtHint != "" && q.ExtHint == ext {
		return 1, "extension"
	}
	for _, tok := range q.Tokens {
		if strings.TrimPrefix(tok, "*.") == ext || strings.TrimPrefix(tok, ".") == ext {
			return 1, "extension"
		}
	}
	return 0, ""
}

// recentlyModifiedRule: exponential decay with a 30-day half-life.
type recentlyModifiedRule struct {
	// now is injectable so tests are deterministic.
	now func() time.Time
}

func (recentlyModifiedRule) ID() string    { return "recently_modified" }
func (recentlyModifiedRule) Label() string { return "Recently modified" }

func (r recentlyModifiedRule) Score(_ Query, c Candidate) (float64, string) {
	now := time.Now()
	if r.now != nil {
		now = r.now()
	}
	days := ageDays(c.MTime, now)
	score := math.Pow(0.5, days/30)
	if score < 0.01 {
		return 0, ""
	}
	return score, "recent"
}

// folderHintRule: fraction of query tokens found in the parent folder path.
type folderHintRule struct{}

func (folderHintRule) ID() string    { return "folder_hint" }
func (folderHintRule) Label() string { return "Folder hint" }

func (folderHintRule) Score(q Query, c Candidate) (float64, string) {
	if len(q.Tokens) == 0 {
		return 0, ""
	}
	dir := strings.ToLower(c.Dir)
	if dir == "" {
		dir = strings.ToLower(filepath.Dir(c.Path))
	}
	if dir == "" {
		return 0, ""
	}
	found := 0
	for _, tok := range q.Tokens {
		if strings.Contains(dir, tok) {
			found++
		}
	}
	if found == 0 {
		return 0, ""
	}
	return float64(found) / float64(len(q.Tokens)), "same folder"
}

// fuzzyNameRule: typo tolerance via bigram Dice (same metric as /ask).
type fuzzyNameRule struct{}

func (fuzzyNameRule) ID() string    { return "fuzzy_name" }
func (fuzzyNameRule) Label() string { return "Fuzzy name" }

func (fuzzyNameRule) Score(q Query, c Candidate) (float64, string) {
	if q.Raw == "" {
		return 0, ""
	}
	score := textmatch.Dice(strings.ToLower(strings.TrimSpace(q.Raw)), stripExt(c.Name))
	if score <= 0 {
		return 0, ""
	}
	return score, "fuzzy"
}
