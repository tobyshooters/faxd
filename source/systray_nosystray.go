//go:build nosystray

package source

import "log"

func StartSystray(d *Daemon, icon []byte) {
	log.Println("systray disabled (built with -tags nosystray)")
}
