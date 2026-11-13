// Command wayerpc runs the native desktop app. --version prints its version.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"wayerpc/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.Current().String())
		return
	}

	app := NewApp()
	screen := newDesktop(app)
	app.notify = screen.notify
	mygo.App.SetName("WayerPC")
	mygo.App.WhenReady(func() {
		// Startup may emit notifications; the queue is drained after the
		// window exists, so no UI access occurs from backend goroutines.
		app.startup(context.Background())
		screen.win = mygo.NewWindow(mygo.WindowOptions{
			Title: "WayerPC", Width: 1180, Height: 720,
			MinWidth: 800, MinHeight: 560, Content: ui.View(screen.view),
		})
		screen.start()
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		screen.stop()
		app.shutdown(context.Background())
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "wayerpc:", err)
		os.Exit(1)
	}
}
