package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"wayerpc/internal/rank"
	"wayerpc/internal/server"
	"wayerpc/internal/syncplan"
)

func TestDesktopNotificationsQueueAndStop(t *testing.T) {
	d := newDesktop(NewApp())
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				d.notify("log", "line")
			}
		}()
	}
	workers.Wait()
	if len(d.queue) != 800 {
		t.Fatalf("queued %d events, want 800", len(d.queue))
	}
	d.stop()
	d.notify("log", "ignored")
	if len(d.queue) != 800 {
		t.Fatal("notification queued after stop")
	}
}

func TestDesktopIgnoresStaleResults(t *testing.T) {
	d := newDesktop(NewApp())
	d.findID = "find-current"
	d.handle(desktopNotification{"find-hits", FindHits{SearchID: "find-old", Hits: []rank.Hit{{}}}})
	if len(d.findHits) != 0 {
		t.Fatal("accepted old search results")
	}
	d.handle(desktopNotification{"find-hits", FindHits{SearchID: "find-current", Hits: []rank.Hit{{}}}})
	if len(d.findHits) != 1 {
		t.Fatal("lost current search results")
	}
	d.syncID = "sync-current"
	d.handle(desktopNotification{"sync-result", SyncScanResult{RunID: "sync-old", Rows: []syncplan.Entry{{Name: "old"}}}})
	if len(d.syncResult.Rows) != 0 {
		t.Fatal("accepted old sync plan")
	}
	d.handle(desktopNotification{"sync-result", SyncScanResult{RunID: "sync-current", Rows: []syncplan.Entry{{Name: "current"}}}})
	if len(d.syncResult.Rows) != 1 {
		t.Fatal("lost current sync plan")
	}
}

// The native view can be exercised without a window or a phone.
func TestDesktopNativePagesRender(t *testing.T) {
	d := newDesktop(NewApp())
	d.rules = rank.DefaultConfig()
	d.weights = make([]string, len(d.rules))
	for i := range d.weights {
		d.weights[i] = "50"
	}
	for _, page := range []string{"Console", "Files", "Find", "Phone", "Sync", "Import", "Rules", "Settings"} {
		d.page = page
		tt := ui.NewTester(d.view, 1180, 720)
		if !tt.HasText(page) {
			t.Errorf("%s page heading missing", page)
		}
	}
}

func TestDesktopShutdownStopsServerWithoutDeadlock(t *testing.T) {
	a := NewApp()
	srv := server.New("127.0.0.1", 0, nil, nil, nil)
	srv.SetObserver(a)
	a.srv = srv
	go func() { _ = srv.Start() }()
	deadline := time.Now().Add(2 * time.Second)
	for !srv.Snapshot().Running && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !srv.Snapshot().Running {
		t.Fatal("server did not start")
	}
	done := make(chan struct{})
	go func() { a.shutdown(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown deadlocked")
	}
}

func TestDesktopSidebarNavigation(t *testing.T) {
	d := newDesktop(NewApp())
	tt := ui.NewTester(d.view, 1180, 720)
	if err := tt.Click("Find"); err != nil {
		t.Fatal(err)
	}
	if d.page != "Find" || !tt.HasText("Search local files") {
		t.Fatalf("navigation failed: page %q", d.page)
	}
	if err := tt.Click("Sync"); err != nil {
		t.Fatal(err)
	}
	if d.page != "Sync" || !tt.HasText("Compare PC and phone") {
		t.Fatalf("navigation failed: page %q", d.page)
	}
}
