package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"wayerpc/internal/catalog"
	"wayerpc/internal/index"
	"wayerpc/internal/rank"
	"wayerpc/internal/session"
	"wayerpc/internal/storage"
	"wayerpc/internal/telemetry"
)

// desktop owns the native UI state. Except for the notification queue, this
// state is only touched on MyGo's UI thread. Slow work runs in goroutines.
type desktop struct {
	app               *App
	win               *mygo.Window
	page              string
	message           string
	status            Status
	catalog           []catalog.Entry
	shared, received  []storage.FileInfo
	logs              []string
	activity          []telemetry.Activity
	transfers         map[string]telemetry.Transfer
	query, phoneQuery string
	findID            string
	findHits          []rank.Hit
	ask               AskPhoneResult
	askBusy           bool
	requests          []session.RequestEntry
	devices           []session.DeviceInfo
	indexState        index.State
	rules             []rank.Config
	weights           []string
	syncID            string
	syncBusy          bool
	syncResult        SyncScanResult
	selected          map[string]bool

	queueMu  sync.Mutex
	queue    []desktopNotification
	wake     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

type desktopNotification struct {
	name string
	data any
}

func newDesktop(app *App) *desktop {
	return &desktop{app: app, page: "Console", transfers: make(map[string]telemetry.Transfer), selected: make(map[string]bool), wake: make(chan struct{}, 1), done: make(chan struct{})}
}

// notify is called by server, search and transfer goroutines. Queue without
// blocking those workers; wakeups are coalesced but individual events are not.
func (d *desktop) notify(name string, values ...any) {
	var value any
	if len(values) > 0 {
		value = values[0]
	}
	d.queueMu.Lock()
	select {
	case <-d.done:
		d.queueMu.Unlock()
		return
	default:
	}
	d.queue = append(d.queue, desktopNotification{name, value})
	d.queueMu.Unlock()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *desktop) start() {
	d.refresh()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-d.done:
				return
			case <-ticker.C:
				st := d.app.GetStatus()
				d.update(func() { d.status = st })
			case <-d.wake:
				d.queueMu.Lock()
				batch := d.queue
				d.queue = nil
				d.queueMu.Unlock()
				d.update(func() {
					for _, event := range batch {
						d.handle(event)
					}
				})
			}
		}
	}()
}

func (d *desktop) stop() { d.stopOnce.Do(func() { close(d.done) }) }

// Update posts to the native UI thread; no UI state is modified by workers.
func (d *desktop) update(fn func()) {
	select {
	case <-d.done:
		return
	default:
	}
	d.win.Update(fn)
}

func (d *desktop) handle(e desktopNotification) {
	switch e.name {
	case "stats":
		if st, ok := e.data.(Status); ok {
			d.status = st
		}
	case "log":
		if line, ok := e.data.(string); ok {
			d.logs = append(d.logs, line)
			if len(d.logs) > 200 {
				d.logs = d.logs[len(d.logs)-200:]
			}
		}
	case "activity":
		if a, ok := e.data.(telemetry.Activity); ok {
			d.activity = append(d.activity, a)
			if len(d.activity) > 100 {
				d.activity = d.activity[len(d.activity)-100:]
			}
		}
	case "transfer":
		if t, ok := e.data.(telemetry.Transfer); ok {
			d.transfers[t.ID] = t
		}
	case "catalog-changed":
		go d.refreshFiles()
	case "find-hits":
		if hits, ok := e.data.(FindHits); ok && hits.SearchID == d.findID {
			d.findHits = append(d.findHits, hits.Hits...)
		}
	case "find-done":
		if result, ok := e.data.(FindDone); ok && result.SearchID == d.findID {
			d.message = fmt.Sprintf("Found %d results in %d ms. %s", result.Total, result.TookMS, result.SkippedNote)
		}
	case "sync-result":
		if result, ok := e.data.(SyncScanResult); ok && result.RunID == d.syncID {
			d.syncResult = result
			d.syncBusy = false
			d.selected = make(map[string]bool)
			if result.Error != "" {
				d.message = result.Error
			} else {
				d.message = "Scan complete. Select files to copy."
			}
		}
	case "sync-progress":
		if p, ok := e.data.(SyncProgress); ok && p.RunID == d.syncID {
			d.message = fmt.Sprintf("Sync %d/%d: %s — %s %s", p.Index, p.Total, p.Name, p.State, p.Error)
		}
	case "index-state":
		if state, ok := e.data.(index.State); ok {
			d.indexState = state
		}
	case "request-update":
		go d.refreshRequests()
	}
}

func (d *desktop) refresh() {
	go func() {
		st, logs, activity, rules, requests := d.app.GetStatus(), d.app.GetLogs(), d.app.GetActivity(), d.app.GetMatchRules(), d.app.GetRequestLog()
		devices, indexState := d.app.GetDevices(), d.app.GetIndexState()
		d.update(func() {
			d.status, d.logs, d.activity, d.rules, d.requests = st, logs, activity, rules, requests
			d.devices, d.indexState = devices, indexState
			d.weights = make([]string, len(rules))
			for i, r := range rules {
				d.weights[i] = fmt.Sprint(r.Weight)
			}
		})
	}()
	go d.refreshFiles()
}

func (d *desktop) refreshFiles() {
	catalog, shared, received := d.app.CatalogList(), d.app.ListShared(), d.app.ListReceived()
	d.update(func() { d.catalog, d.shared, d.received = catalog, shared, received })
}
func (d *desktop) refreshRequests() {
	requests := d.app.GetRequestLog()
	d.update(func() { d.requests = requests })
}

// run keeps blocking operations off the UI thread and displays their result.
func (d *desktop) run(work func() string) {
	go func() { message := work(); d.update(func() { d.message = message }) }()
}

func (d *desktop) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Children(func() {
		ui.Sidebar(c, &d.page, func() {
			for _, name := range []string{"Console", "Files", "Find", "Phone", "Sync", "Import", "Rules", "Settings"} {
				ui.SidebarItem(c.Key(name), name, nil, name)
			}
		}).Width(180)
		ui.Column(c).Grow(1).MinWidth(0).Padding(24).Gap(12).Children(func() {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, d.page).FontSize(26).Bold().Grow(1)
				ui.Text(c, d.app.version.String()).TextColor(t.TextMuted)
			})
			if d.message != "" {
				ui.Text(c, d.message).TextColor(t.TextMuted)
			}
			ui.Divider(c)
			ui.Scroll(c.Key(d.page)).Grow(1).Children(func() {
				ui.Column(c).FillWidth().Gap(12).Children(func() {
					switch d.page {
					case "Console":
						d.console(c)
					case "Files":
						d.files(c)
					case "Find":
						d.find(c)
					case "Phone":
						d.phone(c)
					case "Sync":
						d.syncView(c)
					case "Import":
						d.importView(c)
					case "Rules":
						d.rulesView(c)
					case "Settings":
						d.settings(c)
					}
				})
			})
		})
	})
}

func (d *desktop) console(c *ui.Context) {
	st := d.status
	state := "Stopped"
	if st.Server.Running {
		state = "Running"
	}
	ui.Text(c, "Server: "+state).FontSize(18).Bold()
	ui.Textf(c, "Phone address: %s   |   Listener: %s", st.LanAddr, st.Server.Addr)
	ui.Textf(c, "Connections: %d   |   Sent: %d files   |   Received: %d files", st.Server.Active, st.Server.FilesSent, st.Server.FilesReceived)
	ui.Textf(c, "Phone: %s   |   v2 links: %d", st.Device, st.ControlLinks)
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c, "Start server").Clicked() {
			d.run(d.app.StartServer)
		}
		if ui.Button(c, "Stop server").Clicked() {
			d.run(d.app.StopServer)
		}
		if ui.Button(c, "Refresh").Clicked() {
			d.refresh()
		}
		if ui.Button(c, "Clear logs").Clicked() {
			d.app.ClearLogs()
			d.logs = nil
		}
	})
	ui.Text(c, "Transfers").FontSize(18).Bold()
	for id, transfer := range d.transfers {
		t := transfer
		ui.Row(c.Key(id)).Gap(8).Children(func() {
			ui.Textf(c, "%s: %s — %d / %d bytes (%s)", t.Dir, t.Name, t.Bytes, t.Total, t.State).Grow(1)
			if t.State == telemetry.TransferActive && ui.Button(c, "Cancel").Clicked() {
				d.app.CancelTransfer(t.ID)
			}
		})
	}
	ui.Text(c, "Recent activity").FontSize(18).Bold()
	for i := len(d.activity) - 1; i >= 0 && i >= len(d.activity)-25; i-- {
		a := d.activity[i]
		ui.Textf(c.Key(fmt.Sprint(i)), "%s  %s  %s", a.Kind, a.Detail, a.Status)
	}
	ui.Text(c, "Recent logs").FontSize(18).Bold()
	for i := len(d.logs) - 1; i >= 0 && i >= len(d.logs)-50; i-- {
		ui.Text(c.Key(fmt.Sprint(i)), d.logs[i]).FontSize(12)
	}
}

func (d *desktop) files(c *ui.Context) {
	if ui.Button(c, "Refresh files").Clicked() {
		go d.refreshFiles()
	}
	ui.Text(c, "Catalog").FontSize(18).Bold()
	for _, entry := range d.catalog {
		e := entry
		ui.Row(c.Key("cat:" + e.Path)).Gap(8).Children(func() {
			ui.Text(c, e.Name).Grow(1).SingleLine()
			if ui.Button(c, "Show").Clicked() {
				d.run(func() string { return d.app.RevealInExplorer(e.Path) })
			}
			if ui.Button(c, "Send").Clicked() {
				d.send(e.Path)
			}
			if ui.Button(c, "Remove").Clicked() {
				d.removeCatalog(e.Path)
			}
		})
	}
	for _, group := range []struct {
		name  string
		files []storage.FileInfo
	}{{"Shared", d.shared}, {"Received", d.received}} {
		ui.Text(c, group.name).FontSize(18).Bold()
		for _, file := range group.files {
			f := file
			ui.Row(c.Key(group.name + f.Path)).Gap(8).Children(func() {
				ui.Text(c, f.Name).Grow(1).SingleLine()
				if ui.Button(c, "Show").Clicked() {
					d.run(func() string { return d.app.RevealInExplorer(f.Path) })
				}
				if !f.IsDir && ui.Button(c, "Send").Clicked() {
					d.send(f.Path)
				}
			})
		}
	}
}

func (d *desktop) send(path string) {
	d.run(func() string {
		if err := d.app.SendToPhone(path); err != nil {
			return err.Error()
		}
		return "File offered to phone."
	})
}

func (d *desktop) removeCatalog(path string) {
	d.run(func() string {
		if !confirmAction("Remove from catalog?", path+"\nThe file itself will not be deleted.") {
			return "Cancelled."
		}
		if err := d.app.CatalogRemove(path); err != nil {
			return err.Error()
		}
		return "Removed from catalog."
	})
}

func confirmAction(title, detail string) bool {
	result, err := mygo.Dialog.Message(mygo.MessageOptions{Type: mygo.MessageQuestion, Title: title, Message: title, Detail: detail, Buttons: []string{"Cancel", "Continue"}, CancelButton: 0})
	return err == nil && result.Button == 1
}

func (d *desktop) find(c *ui.Context) {
	ui.Text(c, "Search local files").FontSize(18).Bold()
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c.Key("query"), &d.query).Placeholder("Filename or keywords").Label("Search local files").Grow(1)
		if ui.PrimaryButton(c, "Search").Clicked() && strings.TrimSpace(d.query) != "" {
			if d.findID != "" {
				d.app.CancelFind(d.findID)
			}
			d.findHits = nil
			d.findID = d.app.StartFind(d.query, FindOpts{})
			d.message = "Searching…"
		}
		if ui.Button(c, "Cancel").Clicked() && d.findID != "" {
			d.app.CancelFind(d.findID)
			d.findID = ""
			d.message = "Search cancelled."
		}
	})
	for _, hit := range d.findHits {
		h := hit
		ui.Row(c.Key(h.Path)).Gap(8).Children(func() {
			ui.Textf(c, "%s  (%.0f%%, %s)", h.Name, h.Score*100, h.Reason).Grow(1).SingleLine()
			if ui.Button(c, "Show").Clicked() {
				d.run(func() string { return d.app.RevealInExplorer(h.Path) })
			}
			if ui.Button(c, "Import").Clicked() {
				d.importPaths([]string{h.Path})
			}
			if ui.Button(c, "Send").Clicked() {
				d.send(h.Path)
			}
		})
	}
}

func (d *desktop) phone(c *ui.Context) {
	ui.Text(c, "Find on connected phone").FontSize(18).Bold()
	ui.Text(c, "Requires a connected phone using protocol v2.")
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c.Key("phone-query"), &d.phoneQuery).Placeholder("Filename or keywords").Label("Search phone").Grow(1)
		if ui.PrimaryButton(c, "Ask phone").Disabled(d.askBusy).Clicked() {
			q := d.phoneQuery
			d.askBusy = true
			d.ask = AskPhoneResult{}
			d.message = "Waiting for the phone…"
			go func() {
				result := d.app.AskPhone(q)
				d.update(func() {
					d.ask = result
					d.askBusy = false
					if result.Error != "" {
						d.message = result.Error
					} else {
						d.message = "Phone search: " + result.State
					}
				})
			}()
		}
		if ui.Button(c, "Abort request").Clicked() && d.ask.ID != "" {
			id := d.ask.ID
			d.run(func() string {
				if err := d.app.AbortAsk(id); err != nil {
					return err.Error()
				}
				return "Request aborted."
			})
			d.ask = AskPhoneResult{}
		}
	})
	for _, item := range d.ask.Matches {
		item, id := item, d.ask.ID
		ui.Row(c.Key(fmt.Sprint(item.N))).Gap(8).Children(func() {
			ui.Textf(c, "%s  (%d bytes)", item.Name, item.Size).Grow(1).SingleLine()
			if ui.Button(c, "Request").Clicked() {
				d.run(func() string {
					result, err := d.app.PickAskMatch(id, item.N)
					if err != nil {
						return err.Error()
					}
					return result
				})
			}
		})
	}
	ui.Divider(c)
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, "Phone requests").FontSize(18).Bold().Grow(1)
		if ui.Button(c, "Refresh requests").Clicked() {
			go d.refreshRequests()
		}
	})
	for i := len(d.requests) - 1; i >= 0 && i >= len(d.requests)-50; i-- {
		r := d.requests[i]
		ui.Textf(c.Key(r.ID), "%s: %s — %s (%d matches)", r.Device, r.Query, r.Outcome, r.MatchCount)
	}
}

func (d *desktop) syncView(c *ui.Context) {
	ui.Text(c, "Compare PC and phone").FontSize(18).Bold()
	ui.Text(c, "Select the files to copy. Nothing is deleted; the plan is checked again before copying.")
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c, "Scan").Disabled(d.syncBusy).Clicked() {
			d.syncResult = SyncScanResult{}
			d.selected = make(map[string]bool)
			d.syncBusy = true
			d.syncID = d.app.StartSyncScan()
			d.message = "Scanning…"
		}
		if ui.Button(c, "Copy selected").Disabled(d.syncBusy).Clicked() {
			var names []string
			for _, row := range d.syncResult.Rows {
				if d.selected[row.Name] && row.Action != "none" {
					names = append(names, row.Name)
				}
			}
			if len(names) == 0 {
				d.message = "Select at least one file."
			} else {
				d.syncBusy = true
				d.syncID = d.app.ApplySync(names)
				d.message = "Checking the plan and copying selected files…"
			}
		}
	})
	if d.syncResult.Note != "" {
		ui.Text(c, d.syncResult.Note)
	}
	for _, row := range d.syncResult.Rows {
		row := row
		checked := d.selected[row.Name]
		if ui.Checkbox(c.Key(row.Name), &checked, fmt.Sprintf("%s — %s (%s)", row.Name, row.Verdict, row.Action)).Disabled(d.syncBusy || row.Action == "none").Changed() {
			d.selected[row.Name] = checked
		}
	}
}

func (d *desktop) importPaths(paths []string) {
	d.run(func() string {
		result, err := d.app.CatalogAddPaths(paths)
		if err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Imported %d; skipped %d.", result.Added, result.Skipped)
	})
}

func (d *desktop) importView(c *ui.Context) {
	ui.Text(c, "Add files to the catalog").FontSize(18).Bold()
	ui.Text(c, "Files remain in their original location. You can also import search results from Find.")
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c, "Choose files…").Clicked() {
			go func() {
				paths := d.app.PickFiles()
				if len(paths) > 0 {
					d.importPaths(paths)
				}
			}()
		}
		if ui.Button(c, "Choose folder…").Clicked() {
			d.run(func() string {
				path := d.app.PickFolder("Choose a folder to import")
				if path == "" {
					return "No folder selected."
				}
				result, err := d.app.CatalogAddFolder(path)
				if err != nil {
					return err.Error()
				}
				return fmt.Sprintf("Imported %d; skipped %d.", result.Added, result.Skipped)
			})
		}
	})
}

func (d *desktop) rulesView(c *ui.Context) {
	ui.Text(c, "Search ranking rules").FontSize(18).Bold()
	ui.Text(c, "Enable rules and set weights from 0 to 100.")
	for i := range d.rules {
		i := i
		ui.Row(c.Key(d.rules[i].ID)).Gap(8).Children(func() {
			ui.Checkbox(c, &d.rules[i].Enabled, d.rules[i].ID).Grow(1)
			ui.TextInput(c, &d.weights[i]).Label("Weight for " + d.rules[i].ID).Width(80)
		})
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c, "Save rules").Clicked() {
			rules := append([]rank.Config(nil), d.rules...)
			valid := true
			for i := range rules {
				n, err := strconv.Atoi(strings.TrimSpace(d.weights[i]))
				if err != nil || n < 0 || n > 100 {
					valid = false
					break
				}
				rules[i].Weight = n
			}
			if !valid {
				d.message = "Weights must be whole numbers from 0 to 100."
			} else {
				d.run(func() string {
					if err := d.app.SetMatchRules(rules); err != nil {
						return err.Error()
					}
					return "Rules saved."
				})
			}
		}
		if ui.Button(c, "Reset rules").Clicked() {
			d.run(func() string {
				if err := d.app.ResetMatchRules(); err != nil {
					return err.Error()
				}
				d.refresh()
				return "Rules reset."
			})
		}
	})
}

func (d *desktop) settings(c *ui.Context) {
	ui.Text(c, "Data and connection").FontSize(18).Bold()
	ui.Text(c, "Data folder: "+d.status.DataRoot)
	ui.Text(c, "Shared: "+d.status.Shared)
	ui.Text(c, "Received: "+d.status.Received)
	ui.Textf(c, "Catalog: %d files   |   Data usage: %.2f MB", d.status.CatalogSize, float64(d.status.DiskUsage)/(1024*1024))
	ui.Textf(c, "Search index: %d files   |   Building: %t", d.indexState.Count, d.indexState.Building)
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "Change data folder…").Clicked() {
			d.run(func() string {
				path := d.app.PickFolder("Choose data folder")
				if path == "" {
					return "No folder selected."
				}
				if !confirmAction("Move data folder?", "Existing data will be moved to: "+path) {
					return "Cancelled."
				}
				_, err := d.app.SetDataRoot(path)
				if err != nil {
					return err.Error()
				}
				d.refresh()
				return "Data folder updated."
			})
		}
		if ui.Button(c, "Rebuild search index").Clicked() {
			d.run(d.app.RebuildIndex)
		}
	})
	ui.Text(c, "Known devices").FontSize(18).Bold()
	for _, device := range d.devices {
		ui.Textf(c.Key(device.DID), "%s — %s (connected: %t)", device.Name, device.LastIP, device.Connected)
	}
	if ui.Button(c, "Refresh devices").Clicked() {
		d.refresh()
	}
	enabled := d.status.ProtocolV2
	if ui.Checkbox(c, &enabled, "Enable protocol v2").Changed() {
		d.status.ProtocolV2 = enabled
		d.run(func() string { d.app.SetProtocolV2(enabled); return "Protocol setting updated." })
	}
}
