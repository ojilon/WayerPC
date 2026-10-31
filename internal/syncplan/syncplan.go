// Package syncplan decides what a Sync run would do, without touching a
// single file. It is deliberately pure: callers hand it the PC side
// (received/ + shared/, flattened by name) and the phone side (a
// `scope:"sync"` manifest) and get back one verdict per name.
//
// The rules (see doc/sync.md):
//   - a name present on one side only        → only_pc / only_phone
//   - present on both, same size and the mtimes close (≤ 2 s) or equal sha256
//     → same (nothing to do)
//   - otherwise the newer mtime wins         → newer_pc / newer_phone
//
// Nothing is ever deleted, and `same` is never actioned.
package syncplan

import (
	"sort"
	"strings"
)

// Side is one file as seen on the PC or the phone.
type Side struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	SHA   string `json:"sha256,omitempty"`
}

// Verdicts.
const (
	OnlyPC     = "only_pc"
	OnlyPhone  = "only_phone"
	NewerPC    = "newer_pc"
	NewerPhone = "newer_phone"
	Same       = "same"
)

// Actions.
const (
	ActionPush = "push" // PC → phone
	ActionPull = "pull" // phone → PC
	ActionNone = "none"
)

// MTimesMatch is the closeness that counts as "same version" (file systems
// round mtimes differently, and the phone reports whole seconds).
const MTimesMatch int64 = 2

// Entry is one row of the sync plan.
type Entry struct {
	Name     string `json:"name"`
	Verdict  string `json:"verdict"`
	Action   string `json:"action"`
	PC       *Side  `json:"pc,omitempty"`
	Phone    *Side  `json:"phone,omitempty"`
	DiffSecs int64  `json:"diffSecs,omitempty"`
}

// Plan is the whole comparison.
type Plan struct {
	Entries []Entry `json:"entries"`
	// Counts are conveniences for the UI headings.
	OnlyPC     int   `json:"onlyPc"`
	OnlyPhone  int   `json:"onlyPhone"`
	Different  int   `json:"different"`
	Same       int   `json:"same"`
	TotalBytes int64 `json:"totalBytes"`
}

// FlattenPC collapses received/ + shared/ into one set keyed by name: when a
// name exists in both folders the newer copy wins (the one a phone would
// actually receive).
func FlattenPC(sides []Side) []Side {
	byName := make(map[string]Side, len(sides))
	for _, s := range sides {
		name := cleanName(s.Name)
		if name == "" {
			continue
		}
		s.Name = name
		prev, ok := byName[strings.ToLower(name)]
		if !ok || s.MTime > prev.MTime {
			byName[strings.ToLower(name)] = s
		}
	}
	out := make([]Side, 0, len(byName))
	for _, s := range byName {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// Compare builds the plan. pc and phone may contain duplicates; the newest
// mtime wins on each side.
func Compare(pc, phone []Side) Plan {
	left := index(FlattenPC(pc))
	right := index(FlattenPC(phone))

	names := make([]string, 0, len(left)+len(right))
	for k := range left {
		names = append(names, k)
	}
	for k := range right {
		if _, ok := left[k]; !ok {
			names = append(names, k)
		}
	}
	sort.Strings(names)

	plan := Plan{Entries: make([]Entry, 0, len(names))}
	for _, key := range names {
		p, onPC := left[key]
		ph, onPhone := right[key]
		e := Entry{}
		switch {
		case onPC && !onPhone:
			e = Entry{Name: p.Name, Verdict: OnlyPC, Action: ActionPush}
			plan.OnlyPC++
			plan.TotalBytes += p.Size
		case !onPC && onPhone:
			e = Entry{Name: ph.Name, Verdict: OnlyPhone, Action: ActionPull}
			plan.OnlyPhone++
			plan.TotalBytes += ph.Size
		default:
			name := p.Name
			if name == "" {
				name = ph.Name
			}
			e = Entry{Name: name, PC: &p, Phone: &ph}
			if isSame(p, ph) {
				e.Verdict = Same
				e.Action = ActionNone
				plan.Same++
				break
			}
			plan.Different++
			if p.MTime > ph.MTime {
				e.Verdict = NewerPC
				e.Action = ActionPush
			} else {
				e.Verdict = NewerPhone
				e.Action = ActionPull
			}
			e.DiffSecs = abs(p.MTime - ph.MTime)
			plan.TotalBytes += maxSize(p.Size, ph.Size)
		}
		if e.PC == nil && onPC {
			e.PC = &p
		}
		if e.Phone == nil && onPhone {
			e.Phone = &ph
		}
		plan.Entries = append(plan.Entries, e)
	}
	return plan
}

// isSame implements "same = size + mtime within 2 s, or equal sha256".
func isSame(p, ph Side) bool {
	if p.SHA != "" && ph.SHA != "" && strings.EqualFold(p.SHA, ph.SHA) && p.Size == ph.Size {
		return true
	}
	if p.Size != ph.Size {
		return false
	}
	return abs(p.MTime-ph.MTime) <= MTimesMatch
}

// Select returns the entries for the given names (empty = nothing selected).
func (p Plan) Select(names []string) []Entry {
	if len(names) == 0 {
		return nil
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[strings.ToLower(cleanName(n))] = true
	}
	out := make([]Entry, 0, len(names))
	for _, e := range p.Entries {
		if e.Action == ActionNone {
			continue
		}
		if want[strings.ToLower(e.Name)] {
			out = append(out, e)
		}
	}
	return out
}

// WithAction returns a copy of e with a flipped direction (the UI lets the
// user override an arrow for files that exist on both sides).
func WithAction(e Entry, action string) Entry {
	if e.PC == nil || e.Phone == nil {
		return e // one-sided rows have exactly one possible direction
	}
	if action != ActionPush && action != ActionPull {
		return e
	}
	e.Action = action
	if action == ActionPush {
		e.Verdict = NewerPC
	} else {
		e.Verdict = NewerPhone
	}
	return e
}

func index(sides []Side) map[string]Side {
	out := make(map[string]Side, len(sides))
	for _, s := range sides {
		if s.Name == "" {
			continue
		}
		out[strings.ToLower(s.Name)] = s
	}
	return out
}

// cleanName normalises separators so "a/b.txt" and "a\b.txt" are the same
// flat name (the sync set is flat by design).
func cleanName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxSize(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
