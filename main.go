package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"

	"faxd/source"
)

//go:embed icon.png
var iconData []byte

//go:embed web
var webFS embed.FS

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "run":
		run()
	case "install":
		if err := source.Install(); err != nil {
			log.Fatalf("install: %v", err)
		}
		fmt.Println("faxd installed and started")
	case "uninstall":
		if err := source.Uninstall(); err != nil {
			log.Fatalf("uninstall: %v", err)
		}
		fmt.Println("faxd uninstalled")
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: faxd <run|install|uninstall>")
	os.Exit(1)
}

func run() {
	cfg, err := source.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	d := source.NewDaemon(cfg)

	var webDir fs.FS
	if info, err := os.Stat("web"); err == nil && info.IsDir() {
		log.Println("serving web UI from disk (live reload)")
		webDir = os.DirFS("web")
	} else {
		webDir, _ = fs.Sub(webFS, "web")
	}
	srv := source.StartServer(d, ":7488", webDir)

	go d.Start()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go source.StartSystray(d, iconData)

	<-ctx.Done()
	log.Println("shutting down...")
	d.Shutdown()
	srv.Close()
}
