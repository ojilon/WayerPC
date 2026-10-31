package main

// Sync bridge (phase 6). The comparison itself is pure (internal/syncplan);
// this file only gathers the two sides, then performs the selected transfers
// through the protocol-v2 session layer.
//
// Rules carried over from the spec: the PC set is received/ + shared/
// flattened by name (newer wins), the phone side is a `scope:"sync"` manifest,
// "same" rows are never touched, and nothing is ever deleted.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wayerpc/internal/session"
	"wayerpc/internal/syncplan"
)

// SyncScanResult is the `sync-result` event payload and the call return.
type SyncScanResult struct {
	RunID     string           `json:"runId"`
	Rows      []syncplan.Entry `json:"rows"`
	Note      string           `json:"note,omitempty"`
	Error     string           `json:"error,omitempty"`
	ScannedAt int64            `json:"scannedAt"`
	OnlyPC    int              `json:"onlyPc"`
	OnlyPhone int              `json:"onlyPhone"`
	Different int              `json:"different"`
	Same      int              `json:"same"`
}

// Select narrows the scanned plan to the names the user ticked (same helper
// the pure package offers, so "same" rows can never be selected).
func (r SyncScanResult) Select(names []string) []syncplan.Entry {
	return syncplan.Plan{Entries: r.Rows}.Select(names)
}

// SyncProgress is one `sync-progress` event.
type SyncProgress struct {
	RunID  string `json:"runId"`
	Name   string `json:"name"`
	Action string `json:"action"`
	State  string `json:"state"` // queued | active | done | error | skipped
	Index  int    `json:"index"`
	Total  int    `json:"total"`
	Error  string `json:"error,omitempty"`
}

// pcSide is a sync-plan side plus the path we would actually send from.
type pcSide struct {
	syncplan.Side
	Path string
}

// syncPCFiles flattens received/ + shared/ by name (newer copy wins).
func (a *App) syncPCFiles() ([]pcSide, error) {
	a.mu.Lock()
	store := a.store
	a.mu.Unlock()
	if store == nil {
		return nil, fmt.Errorf("no data folder is open yet")
	}
	var all []pcSide
	for _, dir := range []string{store.ReceivedDir(), store.SharedDir()} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, de := range entries {
			if de.IsDir() || strings.HasPrefix(de.Name(), ".") {
				continue
			}
			info, ierr := de.Info()
			if ierr != nil {
				continue
			}
			all = append(all, pcSide{
				Side: syncplan.Side{
					Name:  de.Name(),
					Size:  info.Size(),
					MTime: info.ModTime().Unix(),
				},
				Path: filepath.Join(dir, de.Name()),
			})
		}
	}
	// Flatten via syncplan so the "newer wins" rule has exactly one home.
	flat := syncplan.FlattenPC(sidesOf(all))
	byName := make(map[string]pcSide, len(all))
	for _, p := range all {
		// Keep the newest path for each flattened name.
		key := strings.ToLower(p.Name)
		prev, ok := byName[key]
		if !ok || p.MTime > prev.MTime {
			byName[key] = p
		}
	}
	out := make([]pcSide, 0, len(flat))
	for _, s := range flat {
		if p, ok := byName[strings.ToLower(s.Name)]; ok {
			p.Side = s
			out = append(out, p)
		}
	}
	return out, nil
}

func sidesOf(in []pcSide) []syncplan.Side {
	out := make([]syncplan.Side, 0, len(in))
	for _, p := range in {
		out = append(out, p.Side)
	}
	return out
}

// pullPhoneManifest asks the phone for its flat sync manifest.
func (a *App) pullPhoneManifest(ctx context.Context) ([]syncplan.Side, error) {
	m := a.currentSession()
	if m == nil || m.ControlLinks() == 0 {
		return nil, fmt.Errorf("No phone connected — the phone has to be on this network for Sync to compare anything")
	}
	items, err := m.Manifest(ctx, "", "sync")
	if err != nil {
		return nil, err
	}
	out := make([]syncplan.Side, 0, len(items))
	for _, it := range items {
		out = append(out, syncplan.Side{Name: it.P, Size: it.Size, MTime: it.MTime, SHA: it.SHA})
	}
	return out, nil
}

// scanSync builds the plan once (used by both the scan and the apply path).
func (a *App) scanSync(ctx context.Context) (SyncScanResult, map[string]pcSide) {
	pc, err := a.syncPCFiles()
	if err != nil {
		return SyncScanResult{Error: err.Error()}, nil
	}
	phone, perr := a.pullPhoneManifest(ctx)
	if perr != nil {
		return SyncScanResult{Error: perr.Error()}, nil
	}
	plan := syncplan.Compare(sidesOf(pc), phone)
	byName := make(map[string]pcSide, len(pc))
	for _, p := range pc {
		byName[strings.ToLower(p.Name)] = p
	}
	res := SyncScanResult{
		Rows:      plan.Entries,
		ScannedAt: time.Now().Unix(),
		OnlyPC:    plan.OnlyPC,
		OnlyPhone: plan.OnlyPhone,
		Different: plan.Different,
		Same:      plan.Same,
	}
	if len(plan.Entries) == 0 {
		res.Note = "Both sides are empty — nothing to compare yet."
	}
	return res, byName
}

// StartSyncScan runs a comparison in the background and emits `sync-result`.
// Returns the run id so the UI can ignore stale results.
func (a *App) StartSyncScan() string {
	a.mu.Lock()
	a.syncSeq++
	runID := fmt.Sprintf("sync-%d", a.syncSeq)
	a.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), v2RequestTimeout)
		defer cancel()
		res, _ := a.scanSync(ctx)
		res.RunID = runID
		a.logf(fmt.Sprintf("[SYNC] Scan %s: %d only on PC, %d only on phone, %d different",
			runID, res.OnlyPC, res.OnlyPhone, res.Different))
		a.emit("sync-result", res)
	}()
	return runID
}

// ApplySync copies the selected names in their planned direction. The scan is
// redone first so a stale list can never move the wrong copy.
func (a *App) ApplySync(names []string) string {
	a.mu.Lock()
	a.syncSeq++
	runID := fmt.Sprintf("sync-%d", a.syncSeq)
	a.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*v2RequestTimeout)
		defer cancel()

		res, byName := a.scanSync(ctx)
		if res.Error != "" {
			a.emit("sync-result", SyncScanResult{RunID: runID, Error: res.Error, ScannedAt: time.Now().Unix()})
			return
		}
		selected := res.Select(names)
		total := len(selected)
		if total == 0 {
			a.logf("[SYNC] Nothing selected — nothing to copy.")
			return
		}
		m := a.currentSession()
		for i, e := range selected {
			progress := SyncProgress{RunID: runID, Name: e.Name, Action: e.Action, Index: i + 1, Total: total, State: "active"}
			a.emit("sync-progress", progress)
			if err := a.copyOne(ctx, m, e, byName); err != nil {
				progress.State = "error"
				progress.Error = err.Error()
				a.emit("sync-progress", progress)
				a.logf(fmt.Sprintf("[SYNC] %s failed: %v", e.Name, err))
				continue
			}
			progress.State = "done"
			a.emit("sync-progress", progress)
		}
		// Re-scan so the UI shows the new state without a manual refresh.
		final, _ := a.scanSync(ctx)
		final.RunID = runID
		a.emit("sync-result", final)
	}()
	return runID
}

// copyOne performs one planned transfer.
func (a *App) copyOne(ctx context.Context, m *session.Manager, e syncplan.Entry, byName map[string]pcSide) error {
	if m == nil || m.ControlLinks() == 0 {
		return fmt.Errorf("no phone connected")
	}
	switch e.Action {
	case syncplan.ActionPush:
		local, ok := byName[strings.ToLower(e.Name)]
		if !ok {
			return fmt.Errorf("no local copy of %s", e.Name)
		}
		ref := session.FileRef{Name: local.Name, Path: local.Path, Size: local.Size, MTime: local.MTime}
		if err := m.Offer(ctx, "", ref); err != nil {
			return err
		}
		a.indexOne(local.Path)
		return nil
	case syncplan.ActionPull:
		ref, err := m.Get(ctx, "", e.Name)
		if err != nil {
			return err
		}
		a.v2Uploaded(ref.Path)
		return nil
	default:
		return nil // "same" is never touched
	}
}
