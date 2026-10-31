// Package rank scores local-search candidates.
//
// A search rule is a tiny, self-contained scoring function: it looks at the
// query and one candidate file and returns a 0..1 score plus a short reason.
// Rules register themselves (see rules.go), each carries a configurable
// weight, and the final score is the weighted mean of the enabled rules.
// Adding a new criterion is one new file plus one Register call — see
// docs/search-rules.md.
package rank

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"wayerpc/internal/textmatch"
)

// DefaultCutoff drops weak matches (mirrors the 20% rule the catalog uses).
const DefaultCutoff = 0.2

// Query is the parsed user input.
type Query struct {
	Raw     string
	Tokens  []string
	ExtHint string // last token that looks like a bare extension ("pdf")
}

// NewQuery parses raw search text.
func NewQuery(raw string) Query {
	q := Query{Raw: strings.TrimSpace(raw)}
	q.Tokens = textmatch.Tokens(q.Raw)
	if n := len(q.Tokens); n > 0 {
		last := strings.TrimPrefix(q.Tokens[n-1], ".")
		if isExtLike(last) {
			q.ExtHint = last
		}
	}
	return q
}

// isExtLike reports whether tok looks like a file extension (letters/digits).
func isExtLike(tok string) bool {
	if len(tok) < 2 || len(tok) > 5 {
		return false
	}
	for _, r := range tok {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// Candidate is one file the engine may score.
type Candidate struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Dir   string `json:"dir,omitempty"`
	Ext   string `json:"ext,omitempty"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"` // unix seconds
}

// Rule scores one candidate against one query. Score is clamped to 0..1.
type Rule interface {
	ID() string
	Label() string
	Score(q Query, c Candidate) (float64, string)
}

// Config is one rule's persisted configuration (setting `match_rules`).
type Config struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	Weight  int    `json:"weight"` // 0..100
}

// defaultWeights are the shipped defaults, in tie-break order.
var defaultWeights = []struct {
	id     string
	weight int
}{
	{"exact_name", 100},
	{"name_contains", 78},
	{"extension", 55},
	{"recently_modified", 42},
	{"folder_hint", 28},
	{"fuzzy_name", 36},
}

// DefaultConfig returns the shipped rule list (all enabled).
func DefaultConfig() []Config {
	out := make([]Config, 0, len(defaultWeights))
	for _, d := range defaultWeights {
		out = append(out, Config{ID: d.id, Enabled: true, Weight: d.weight})
	}
	return out
}

// ParseConfig decodes the persisted JSON, falling back to the defaults for
// anything missing or malformed (a corrupt setting must never break search).
func ParseConfig(raw string) []Config {
	if strings.TrimSpace(raw) == "" {
		return DefaultConfig()
	}
	var cfg []Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultConfig()
	}
	return NormalizeConfig(cfg)
}

// NormalizeConfig keeps known rules, drops duplicates/unknown ids, clamps
// weights and appends any registered rule the caller forgot.
func NormalizeConfig(cfg []Config) []Config {
	seen := map[string]bool{}
	out := make([]Config, 0, len(cfg)+len(registry))
	for _, c := range cfg {
		if _, ok := registry[c.ID]; !ok || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		w := c.Weight
		if w < 0 {
			w = 0
		}
		if w > 100 {
			w = 100
		}
		out = append(out, Config{ID: c.ID, Enabled: c.Enabled, Weight: w})
	}
	for _, r := range registry {
		if seen[r.ID()] {
			continue
		}
		weight := 50
		for _, d := range defaultWeights {
			if d.id == r.ID() {
				weight = d.weight
			}
		}
		out = append(out, Config{ID: r.ID(), Enabled: true, Weight: weight})
	}
	return out
}

// MarshalConfig serialises a config list for the settings table.
func MarshalConfig(cfg []Config) (string, error) {
	b, err := json.Marshal(NormalizeConfig(cfg))
	return string(b), err
}

// Engine ranks candidates with one config snapshot.
type Engine struct {
	bound []boundRule
	cfg   []Config
}

type boundRule struct {
	rule   Rule
	weight float64
}

// NewEngine builds an engine from a config list (order = tie-break order).
func NewEngine(cfg []Config) *Engine {
	cfg = NormalizeConfig(cfg)
	e := &Engine{cfg: cfg}
	for _, c := range cfg {
		rule, ok := registry[c.ID]
		if !ok || !c.Enabled || c.Weight == 0 {
			continue
		}
		e.bound = append(e.bound, boundRule{rule: rule, weight: float64(c.Weight)})
	}
	return e
}

// Config returns the normalised config the engine was built with.
func (e *Engine) Config() []Config { return append([]Config(nil), e.cfg...) }

// Hit is one scored candidate.
type Hit struct {
	Candidate
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// Rank scores every candidate, drops anything below cutoff, sorts by score
// (config order breaks ties) and returns at most limit hits.
func (e *Engine) Rank(q Query, cands []Candidate, cutoff float64, limit int) []Hit {
	if cutoff <= 0 {
		cutoff = DefaultCutoff
	}
	if limit <= 0 {
		limit = 50
	}
	type scored struct {
		hit    Hit
		vector []float64
	}
	rows := make([]scored, 0, len(cands))
	for _, c := range cands {
		score, reason, vector := e.score(q, c)
		if score < cutoff {
			continue
		}
		rows = append(rows, scored{hit: Hit{Candidate: c, Score: round2(score), Reason: reason}, vector: vector})
	}
	// Score first; on a tie the per-rule vector in config order decides, so
	// dragging a rule to the top makes it the visible tie-breaker.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].hit.Score != rows[j].hit.Score {
			return rows[i].hit.Score > rows[j].hit.Score
		}
		vi, vj := rows[i].vector, rows[j].vector
		for k := 0; k < len(vi) && k < len(vj); k++ {
			if vi[k] != vj[k] {
				return vi[k] > vj[k]
			}
		}
		return rows[i].hit.Name < rows[j].hit.Name
	})
	hits := make([]Hit, 0, len(rows))
	for _, r := range rows {
		hits = append(hits, r.hit)
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// score returns the weighted mean plus the strongest rule's label, and the
// per-rule vector (config order) used for tie-breaking.
func (e *Engine) score(q Query, c Candidate) (float64, string, []float64) {
	if len(e.bound) == 0 {
		return 0, "", nil
	}
	vector := make([]float64, 0, len(e.bound))
	var sum, weights float64
	bestContrib := -1.0
	reason := ""
	for _, b := range e.bound {
		raw, why := b.rule.Score(q, c)
		if raw < 0 {
			raw = 0
		}
		if raw > 1 {
			raw = 1
		}
		vector = append(vector, raw)
		sum += raw * b.weight
		weights += b.weight
		if contrib := raw * b.weight; contrib > bestContrib {
			bestContrib = contrib
			reason = why
		}
	}
	if weights == 0 {
		return 0, "", vector
	}
	return sum / weights, reason, vector
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// ---------- registry ----------

var (
	registry = map[string]Rule{}
	order    []string
)

// Register adds a rule to the global registry. Rules self-register from
// their init(); calling it twice with the same id panics (a duplicate id is a
// programming error, not a runtime condition).
func Register(r Rule) {
	if r == nil || r.ID() == "" {
		panic("rank: rule with empty id")
	}
	if _, exists := registry[r.ID()]; exists {
		panic(fmt.Sprintf("rank: duplicate rule id %q", r.ID()))
	}
	registry[r.ID()] = r
	order = append(order, r.ID())
}

// Lookup returns a registered rule by id.
func Lookup(id string) (Rule, bool) {
	r, ok := registry[id]
	return r, ok
}

// Registered lists every registered rule in registration order (tests + UI).
func Registered() []Rule {
	out := make([]Rule, 0, len(order))
	for _, id := range order {
		out = append(out, registry[id])
	}
	return out
}

// ---------- shared rule helpers ----------

// stripExt drops the extension (".pdf") from a filename and lowercases it.
func stripExt(name string) string {
	lower := strings.ToLower(name)
	if i := strings.LastIndexByte(lower, '.'); i > 0 {
		return lower[:i]
	}
	return lower
}

// ageDays is how old a file is, in days, never negative.
func ageDays(mtime int64, now time.Time) float64 {
	if mtime <= 0 {
		return 3650 // unknown timestamp: treat as ancient
	}
	d := now.Sub(time.Unix(mtime, 0)).Hours() / 24
	if d < 0 {
		return 0
	}
	return d
}
