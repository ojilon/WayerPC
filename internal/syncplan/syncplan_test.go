package syncplan

import "testing"

func byName(plan Plan) map[string]Entry {
	out := make(map[string]Entry, len(plan.Entries))
	for _, e := range plan.Entries {
		out[e.Name] = e
	}
	return out
}

func TestCompareVerdicts(t *testing.T) {
	pc := []Side{
		{Name: "only-pc.pdf", Size: 10, MTime: 100},
		{Name: "newer-pc.docx", Size: 10, MTime: 500},
		{Name: "newer-phone.pdf", Size: 10, MTime: 100},
		{Name: "same-size-time.txt", Size: 10, MTime: 1000},
		{Name: "same-sha.bin", Size: 20, MTime: 100, SHA: "ABC"},
		{Name: "size-differs.txt", Size: 10, MTime: 100},
	}
	phone := []Side{
		{Name: "only-phone.jpg", Size: 99, MTime: 50},
		{Name: "newer-pc.docx", Size: 10, MTime: 400},
		{Name: "newer-phone.pdf", Size: 10, MTime: 900},
		{Name: "same-size-time.txt", Size: 10, MTime: 1000 + MTimesMatch},
		{Name: "same-sha.bin", Size: 20, MTime: 9000, SHA: "abc"},
		{Name: "size-differs.txt", Size: 11, MTime: 100},
	}
	plan := Compare(pc, phone)
	got := byName(plan)

	cases := []struct {
		name    string
		verdict string
		action  string
	}{
		{"only-pc.pdf", OnlyPC, ActionPush},
		{"only-phone.jpg", OnlyPhone, ActionPull},
		{"newer-pc.docx", NewerPC, ActionPush},
		{"newer-phone.pdf", NewerPhone, ActionPull},
		{"same-size-time.txt", Same, ActionNone},
		{"same-sha.bin", Same, ActionNone},
		{"size-differs.txt", NewerPhone, ActionPull}, // equal mtimes, equal size? no → newer side wins by mtime tie → phone
	}
	for _, c := range cases {
		e, ok := got[c.name]
		if !ok {
			t.Fatalf("%s missing from the plan", c.name)
		}
		if e.Verdict != c.verdict || e.Action != c.action {
			t.Fatalf("%s = %s/%s, want %s/%s", c.name, e.Verdict, e.Action, c.verdict, c.action)
		}
	}
	if plan.OnlyPC != 1 || plan.OnlyPhone != 1 || plan.Different != 3 || plan.Same != 2 {
		t.Fatalf("counts = %+v", plan)
	}
}

func TestMTimesWithinTwoSecondsCountAsSame(t *testing.T) {
	pc := []Side{{Name: "a.txt", Size: 5, MTime: 1000}}
	for _, delta := range []int64{0, 1, -1, 2, -2} {
		phone := []Side{{Name: "a.txt", Size: 5, MTime: 1000 + delta}}
		if p := Compare(pc, phone); p.Entries[0].Verdict != Same {
			t.Fatalf("delta %d = %s, want same", delta, p.Entries[0].Verdict)
		}
	}
	for _, delta := range []int64{3, -3, 60} {
		phone := []Side{{Name: "a.txt", Size: 5, MTime: 1000 + delta}}
		if p := Compare(pc, phone); p.Entries[0].Verdict == Same {
			t.Fatalf("delta %d counted as same", delta)
		}
	}
}

func TestSameSHAWinsOverMTime(t *testing.T) {
	pc := []Side{{Name: "a.bin", Size: 5, MTime: 10, SHA: "deadbeef"}}
	phone := []Side{{Name: "a.bin", Size: 5, MTime: 99999, SHA: "DEADBEEF"}}
	if p := Compare(pc, phone); p.Entries[0].Verdict != Same {
		t.Fatalf("equal sha treated as %s", p.Entries[0].Verdict)
	}
	// A different hash with the same size is still "different" (newest wins).
	phone[0].SHA = "cafebabe"
	phone[0].MTime = 99999
	if p := Compare(pc, phone); p.Entries[0].Verdict != NewerPhone {
		t.Fatalf("different sha = %s, want newer_phone", p.Entries[0].Verdict)
	}
}

func TestFlattenPCNewerWinsAcrossFolders(t *testing.T) {
	sides := []Side{
		{Name: "dup.txt", Size: 1, MTime: 100}, // received/
		{Name: "dup.txt", Size: 2, MTime: 300}, // shared/ — newer
		{Name: "solo.txt", Size: 3, MTime: 50},
	}
	flat := FlattenPC(sides)
	if len(flat) != 2 {
		t.Fatalf("flatten = %+v", flat)
	}
	for _, s := range flat {
		if s.Name == "dup.txt" && (s.Size != 2 || s.MTime != 300) {
			t.Fatalf("flatten kept the older copy: %+v", s)
		}
	}
}

func TestFlattenPCCaseInsensitiveAndSeparators(t *testing.T) {
	sides := []Side{
		{Name: "Report.PDF", Size: 1, MTime: 10},
		{Name: "report.pdf", Size: 1, MTime: 20},
		{Name: "sub/dir/notes.txt", Size: 2, MTime: 5},
		{Name: `deep\win\path.txt`, Size: 2, MTime: 6},
	}
	flat := FlattenPC(sides)
	names := map[string]bool{}
	for _, s := range flat {
		names[s.Name] = true
	}
	if len(flat) != 3 {
		t.Fatalf("flatten = %+v", flat)
	}
	if !names["notes.txt"] || !names["path.txt"] {
		t.Fatalf("separators not flattened: %+v", flat)
	}
}

func TestNeverDeletes(t *testing.T) {
	pc := []Side{{Name: "a", Size: 1, MTime: 1}, {Name: "b", Size: 1, MTime: 1}}
	phone := []Side{{Name: "b", Size: 1, MTime: 9}, {Name: "c", Size: 1, MTime: 1}}
	for _, e := range Compare(pc, phone).Entries {
		if e.Action != ActionPush && e.Action != ActionPull && e.Action != ActionNone {
			t.Fatalf("unexpected action %q", e.Action)
		}
	}
	// Every one-sided name gets a direction; nothing is ever "delete".
	if p := Compare(pc, phone); len(p.Entries) != 3 {
		t.Fatalf("the plan must mention every name: %+v", p.Entries)
	}
}

func TestSelectAndFlip(t *testing.T) {
	pc := []Side{{Name: "both.txt", Size: 1, MTime: 100}, {Name: "solo.txt", Size: 1, MTime: 100}}
	phone := []Side{{Name: "both.txt", Size: 1, MTime: 200}}
	plan := Compare(pc, phone)

	sel := plan.Select([]string{"both.txt", "solo.txt", "missing.txt"})
	if len(sel) != 2 {
		t.Fatalf("select = %+v", sel)
	}
	// "same" rows are never selectable.
	plan2 := Compare(phone, []Side{{Name: "both.txt", Size: 1, MTime: 201}})
	if got := plan2.Select([]string{"both.txt"}); len(got) != 0 {
		t.Fatalf("same row was selectable: %+v", got)
	}

	flipped := WithAction(sel[0], ActionPush)
	if flipped.Action != ActionPush || flipped.Verdict != NewerPC {
		t.Fatalf("flip = %+v", flipped)
	}
	// A one-sided row keeps its only possible direction.
	solo := plan.Select([]string{"solo.txt"})[0]
	if v := WithAction(solo, ActionPull); v.Action != ActionPush {
		t.Fatalf("one-sided row flipped to %q", v.Action)
	}
}

func TestTotalBytesCountsWorkNotDuplicates(t *testing.T) {
	pc := []Side{{Name: "a", Size: 100, MTime: 1}, {Name: "same", Size: 500, MTime: 1}}
	phone := []Side{{Name: "b", Size: 200, MTime: 1}, {Name: "same", Size: 500, MTime: 1}}
	plan := Compare(pc, phone)
	if plan.TotalBytes != 300 {
		t.Fatalf("totalBytes = %d, want 300 (identical files do not count)", plan.TotalBytes)
	}
}
