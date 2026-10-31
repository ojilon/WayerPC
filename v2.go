package main

// Protocol v2 bridge: the session manager (internal/session) answers "what
// does this PC have", "where do uploads go" and "tell the UI" through Deps.
// Everything phone-facing here is additive — the v1 bridge surface is
// untouched, and the frontend feature-detects the new methods.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wayerpc/internal/config"
	"wayerpc/internal/session"
	"wayerpc/internal/telemetry"
)

// v2RequestTimeout bounds one Ask/Pick call at the bridge boundary. It is a
// touch over the protocol TTL (120 s) so the wire times out first and the UI
// gets the protocol's own error instead of a bare context deadline.
const v2RequestTimeout = 130 * time.Second

// AskPhoneResult is what the Ask-phone stepper renders after step 1.
type AskPhoneResult struct {
	ID       string         `json:"id"`
	State    string         `json:"state"` // matches | none | error
	Query    string         `json:"query"`
	Matches  []session.Item `json:"matches,omitempty"`
	Error    string         `json:"error,omitempty"`
	Device   string         `json:"device,omitempty"`
	ElapsedM int64          `json:"elapsedMs"`
}

// V2State is the protocol-v2 status shown in Settings + the title bar.
type V2State struct {
	Enabled       bool   `json:"enabled"`
	Protocol      int    `json:"protocolVersion"`
	ControlLinks  int    `json:"controlLinks"`
	ConnectedName string `json:"connectedName,omitempty"`
}

// AskUpdate is the payload of the `ask-update` event (Ask-phone stepper).
type AskUpdate struct {
	ID         string `json:"id,omitempty"`
	State      string `json:"state"`
	Query      string `json:"query,omitempty"`
	MatchCount int    `json:"matchCount,omitempty"`
	Picked     int    `json:"picked,omitempty"`
	Name       string `json:"name,omitempty"`
	Path       string `json:"path,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ---------- manager lifecycle ----------

// sessionManager returns the (lazily created) v2 session manager.
func (a *App) sessionManager() *session.Manager {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sess == nil {
		a.sess = session.New(session.Deps{
			Logf:         a.logf,
			Activity:     a.v2Activity,
			Transfer:     func(t telemetry.Transfer) { a.Transfer(t) },
			LocalMatches: a.v2LocalMatches,
			ResolveName:  a.v2ResolveName,
			ReceiveDir:   a.v2ReceiveDir,
			Uploaded:     a.v2Uploaded,
			Hash:         a.v2Hash,
			DeviceSeen:   a.v2DeviceSeen,
			PCName:       a.v2PCName(),
		})
		a.sess.OnRequest = func(e session.RequestEntry) {
			a.emit("request-update", e)
		}
	}
	return a.sess
}

func (a *App) v2Activity(kind, detail, status, device string) {
	a.pushActivity(telemetry.Activity{
		TS:     time.Now().Unix(),
		Kind:   kind,
		Detail: detail,
		Status: status,
		Device: device,
	})
}

// v2PCName is the friendly PC name advertised in hello_ok.
func (a *App) v2PCName() string {
	if name, err := os.Hostname(); err == nil && name != "" {
		return name
	}
	return "WayerPC"
}

// V2Enabled reports the persisted setting (default on).
func (a *App) V2Enabled() bool {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	return cfg.ProtocolV2
}

// GetV2State is used by Settings and the phone screens.
func (a *App) GetV2State() V2State {
	st := V2State{Enabled: a.V2Enabled(), Protocol: 2}
	if m := a.currentSession(); m != nil {
		st.ControlLinks = m.ControlLinks()
		if dev, ok := m.ConnectedDevice(); ok {
			st.ConnectedName = dev.Name
		}
	}
	return st
}

func (a *App) currentSession() *session.Manager {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sess
}

// ---------- Deps implementations ----------

// v2ReceiveDir is where phone pushes land.
func (a *App) v2ReceiveDir() string {
	a.mu.Lock()
	store := a.store
	a.mu.Unlock()
	if store == nil {
		return ""
	}
	return store.ReceivedDir()
}

// v2Uploaded mirrors the v1 post-upload bookkeeping: index + catalog + UI.
func (a *App) v2Uploaded(path string) {
	if path == "" {
		return
	}
	a.indexOne(path)
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat != nil {
		_, _ = cat.AddPath(path)
	}
	a.emit("catalog-changed")
	a.invalidateDiskUsage()
}

// v2Hash streams sha256 and remembers it in the hash_cache table so the same
// file is never hashed twice (the cache is keyed by path + size + mtime).
func (a *App) v2Hash(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat != nil {
		if sha, ok := cat.CachedHash(path, st.Size(), st.ModTime().Unix()); ok {
			return sha, nil
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 256*1024)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", err
	}
	sha := hex.EncodeToString(h.Sum(nil))
	if cat != nil {
		_ = cat.PutHash(path, sha, st.Size(), st.ModTime().Unix())
	}
	return sha, nil
}

// v2DeviceSeen records a phone handshake so Devices survives restarts.
func (a *App) v2DeviceSeen(did, name, ip string) {
	if did == "" {
		return
	}
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat != nil {
		_ = cat.UpsertDevice(did, name, ip, time.Now().Unix())
	}
}

// v2Ref stats one path into the session's FileRef shape.
func v2Ref(path string) (session.FileRef, bool) {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return session.FileRef{}, false
	}
	return session.FileRef{
		Name:  filepath.Base(path),
		Path:  path,
		Size:  st.Size(),
		MTime: st.ModTime().Unix(),
	}, true
}

// v2LocalMatches answers a phone query from the local index (phase 2), with a
// live-walk fallback so a fresh install can still answer the very first query.
func (a *App) v2LocalMatches(query string, max int) []session.FileRef {
	if max <= 0 || max > 50 {
		max = 50
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	hits := a.searchIndex(q, max)
	if len(hits) == 0 {
		hits = a.liveSearch(q, max)
	}
	out := make([]session.FileRef, 0, len(hits))
	seen := make(map[string]bool, len(hits))
	for _, hit := range hits {
		if seen[hit.Path] {
			continue
		}
		seen[hit.Path] = true
		if ref, ok := v2Ref(hit.Path); ok {
			out = append(out, ref)
		}
		if len(out) >= max {
			break
		}
	}
	return out
}

// v2ResolveName maps a phone-supplied bare name to a PC file: received/ and
// shared/ first (the folders the phone is allowed to ask about), then the
// catalog for imported files.
func (a *App) v2ResolveName(name string) (session.FileRef, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return session.FileRef{}, false
	}
	base := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if base == "" || base == "." || strings.HasPrefix(base, ".") {
		return session.FileRef{}, false
	}
	a.mu.Lock()
	store := a.store
	cat := a.cat
	a.mu.Unlock()
	if store != nil {
		for _, dir := range []string{store.ReceivedDir(), store.SharedDir()} {
			if ref, ok := v2Ref(filepath.Join(dir, base)); ok {
				return ref, true
			}
		}
	}
	if cat != nil {
		if entries, err := cat.List(); err == nil {
			for _, e := range entries {
				if strings.EqualFold(filepath.Base(e.Path), base) {
					if ref, ok := v2Ref(e.Path); ok {
						return ref, true
					}
				}
			}
		}
	}
	return session.FileRef{}, false
}

// SetProtocolV2 turns the opt-in protocol on or off (persisted) and rebinds
// the live server without a restart.
func (a *App) SetProtocolV2(enabled bool) V2State {
	a.mu.Lock()
	a.cfg.ProtocolV2 = enabled
	cfg := a.cfg
	srv := a.srv
	a.mu.Unlock()
	if err := config.Save(cfg); err != nil {
		a.logf(fmt.Sprintf("Could not save the protocol setting: %v", err))
	}
	if srv != nil {
		if enabled {
			srv.SetV2(a.sessionManager())
		} else {
			srv.SetV2(nil)
		}
	}
	if enabled {
		a.logf("Protocol v2 enabled — the phone may use the new two-way link.")
	} else {
		a.logf("Protocol v2 disabled — only the classic phone protocol is served.")
	}
	return a.GetV2State()
}

// ---------- bridge: PC → phone ----------

// AskPhone runs step 1 of the Ask-phone flow: send the query, collect matches.
func (a *App) AskPhone(query string) AskPhoneResult {
	q := strings.TrimSpace(query)
	res := AskPhoneResult{Query: q, State: "error"}
	if q == "" {
		res.Error = "Type something to look for on the phone."
		return res
	}
	m := a.currentSession()
	if m == nil || m.ControlLinks() == 0 {
		res.Error = "No phone connected — open the app on your phone and join this network first."
		return res
	}
	a.emit("ask-update", AskUpdate{State: "querying", Query: q})
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), v2RequestTimeout)
	defer cancel()
	id, items, err := m.Ask(ctx, "", q, 50)
	res.ID = id
	res.ElapsedM = time.Since(started).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		a.emit("ask-update", AskUpdate{ID: id, State: "error", Query: q, Error: res.Error})
		return res
	}
	res.Matches = items
	if len(items) == 0 {
		res.State = "none"
	} else {
		res.State = "matches"
	}
	a.emit("ask-update", AskUpdate{ID: id, State: res.State, Query: q, MatchCount: len(items)})
	return res
}

// PickAskMatch runs step 2: ask the phone for match n and wait for the bytes.
// The returned string is the saved path inside received/.
func (a *App) PickAskMatch(id string, n int) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("no request in progress")
	}
	m := a.currentSession()
	if m == nil {
		return "", errors.New("protocol v2 is not active")
	}
	if n <= 0 {
		return "", errors.New("pick a match number first")
	}
	a.emit("ask-update", AskUpdate{ID: id, State: "picking", Picked: n})
	ctx, cancel := context.WithTimeout(context.Background(), v2RequestTimeout)
	defer cancel()
	ref, err := m.Pick(ctx, "", id, n)
	if err != nil {
		a.emit("ask-update", AskUpdate{ID: id, State: "error", Picked: n, Error: err.Error()})
		return "", err
	}
	a.v2Uploaded(ref.Path)
	a.emit("ask-update", AskUpdate{ID: id, State: "sent", Picked: n, Name: ref.Name, Path: ref.Path})
	return ref.Path, nil
}

// AbortAsk cancels an in-flight ask on the phone side.
func (a *App) AbortAsk(id string) error {
	m := a.currentSession()
	if m == nil {
		return errors.New("protocol v2 is not active")
	}
	if err := m.Abort("", id); err != nil {
		return err
	}
	a.emit("ask-update", AskUpdate{ID: id, State: "aborted"})
	return nil
}

// SendToPhone offers one PC file to the connected phone (Files → Send).
func (a *App) SendToPhone(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("choose a file to send")
	}
	ref, ok := v2Ref(path)
	if !ok {
		return fmt.Errorf("cannot read %s", filepath.Base(path))
	}
	m := a.currentSession()
	if m == nil || m.ControlLinks() == 0 {
		return errors.New("No phone connected — the phone has to be on this network for Send to work.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), v2RequestTimeout)
	defer cancel()
	if err := m.Offer(ctx, "", ref); err != nil {
		a.emit("ask-update", AskUpdate{State: "error", Name: ref.Name, Path: ref.Path, Error: err.Error()})
		return err
	}
	a.emit("ask-update", AskUpdate{State: "sent", Name: ref.Name, Path: ref.Path})
	a.note(telemetry.KindFileSent, ref.Name, "Complete")
	return nil
}

// GetRequestLog returns the phone-initiated request history (newest last).
func (a *App) GetRequestLog() []session.RequestEntry {
	m := a.currentSession()
	if m == nil {
		return nil
	}
	return m.Requests()
}

// GetDevices merges the persisted device registry with the live link state.
func (a *App) GetDevices() []session.DeviceInfo {
	live := map[string]session.DeviceInfo{}
	if m := a.currentSession(); m != nil {
		for _, d := range m.Devices() {
			live[d.DID] = d
		}
	}
	out := make([]session.DeviceInfo, 0, len(live))
	for _, d := range live {
		out = append(out, d)
	}
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat != nil {
		if rows, err := cat.Devices(); err == nil {
			for _, row := range rows {
				if _, ok := live[row.DID]; ok {
					continue // live rows already carry connected=true
				}
				out = append(out, session.DeviceInfo{
					DID: row.DID, Name: row.Name, LastSeen: row.LastSeen, LastIP: row.LastIP,
				})
			}
		}
	}
	return out
}
