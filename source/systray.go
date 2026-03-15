//go:build !nosystray

package source

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/getlantern/systray"
)

func StartSystray(d *Daemon, icon []byte) {
	systray.Run(func() { onReady(d, icon) }, func() {})
}

func onReady(d *Daemon, icon []byte) {
	if len(icon) > 0 {
		systray.SetIcon(icon)
	}
	systray.SetTitle("faxd")
	systray.SetTooltip("faxd - local fax daemon")

	mStatus := systray.AddMenuItem("faxd running", "")
	mStatus.Disable()
	mLast := systray.AddMenuItem("Last fax: never", "")
	mLast.Disable()

	systray.AddSeparator()

	mOpen := systray.AddMenuItem("Open Web UI", "")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "")

	go func() {
		for {
			time.Sleep(5 * time.Second)
			_, last := d.Status()
			if last.IsZero() {
				mLast.SetTitle("Last fax: never")
			} else {
				ago := time.Since(last).Truncate(time.Second)
				mLast.SetTitle(fmt.Sprintf("Last fax: %s ago", ago))
			}
		}
	}()

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser("http://localhost:8080")
			case <-mQuit.ClickedCh:
				d.Shutdown()
				systray.Quit()
			}
		}
	}()
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}
