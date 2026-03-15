# faxd — Project Spec

## Overview
A local daemon written in Go that simulates a fax machine using email (IMAP).
It polls a dedicated Gmail inbox, filters by sender whitelist, and prints
attachments to the local printer. A systray icon and localhost web UI provide
the interface.

## Stack
- **Language:** Go
- **Systray:** `github.com/getlantern/systray`
- **Web UI:** Go `net/http` + embedded HTML/JS (`//go:embed`)
- **IMAP:** `github.com/emersion/go-imap`
- **Printing:** `lp`/`lpr` subprocess (macOS/Linux)
- **Config:** TOML at `~/.config/faxd/config.toml`

## CLI
```
faxd run        # start daemon + web UI + systray
faxd install    # write launchd (macOS) or systemd user service (Linux) and start
faxd uninstall  # remove service
```

## Config file
```toml
email = "myfax@gmail.com"
password = "gmail-app-password"
poll_interval_seconds = 30
allowed_senders = ["alice@gmail.com", "bob@gmail.com"]
max_attachment_mb = 20
allowed_extensions = [".pdf", ".jpg", ".jpeg", ".png"]
monochrome = true
scaling = 50
fit_to_page = false
```
Created automatically with defaults if it doesn't exist.

## Project structure
```
faxd/
  main.go              # CLI entrypoint, subcommands, embeds web/ and icon.png
  icon.png             # systray + favicon icon
  web/
    index.html         # single page UI
    app.js
    style.css
    favicon.png
  source/
    config.go          # load/save config.toml
    daemon.go          # poll loop, debug log ring buffer, orchestrates imap + print
    imap.go            # IMAP connection, fetch unseen, filter whitelist, base64 decode
    print.go           # dither images, shell out to lp with configurable options
    server.go          # :8080 HTTP server, JSON API
    install.go         # write launchd plist or systemd unit file
    systray.go         # systray icon + menu (build tag: default)
    systray_nosystray.go # no-op fallback (build tag: nosystray)
```

## Web UI pages (tabs, persisted via URL hash)
- **Dashboard** — status (running/idle), last fax received, debug log (auto-refreshes every 3s)
- **Inbox log** — list of received faxes (sender, time, filename, status) (auto-refreshes every 3s)
- **Settings** — edit config (email, poll interval, print options: monochrome/scaling/fit-to-page) + manage allowed senders
- **Send** — upload a PDF/image (with image preview) and enter a recipient email address

## Web API (JSON)
```
GET  /api/status       # daemon status, last activity
GET  /api/log          # received fax history
GET  /api/debug        # debug log lines (ring buffer, last 200)
GET  /api/config       # current config (password stripped)
POST /api/config       # update config (merges with existing)
POST /api/send         # send fax (multipart: file + to address)
```

## Fax receive flow
1. Poll IMAP inbox every N seconds
2. Fetch unseen emails
3. Check sender against whitelist — ignore if not allowed
4. Check attachment extension and size
5. Decode base64 content-transfer-encoding
6. Save attachment to `~/.local/share/faxd/received/`
7. Dither images (Floyd-Steinberg, monochrome, 800px max) if jpg/jpeg/png
8. Shell out to `lp` with configured options (monochrome, scaling, fit-to-page)
9. Mark email as read
10. Log to in-memory + persisted JSON log
11. Update systray tooltip

## Systray menu
```
● faxd running
Last fax: 10 mins ago
─────────────
Open Web UI
─────────────
Quit
```

## Platform notes
- macOS: launchd plist at `~/Library/LaunchAgents/com.faxd.plist`
- Linux: systemd unit at `~/.config/systemd/user/faxd.service`
- Systray on Linux requires `libayatana-appindicator3-dev` and a compatible
  desktop environment (KDE/XFCE natively; GNOME needs AppIndicator extension)
- Build without systray: `go build -tags nosystray`
- Image dithering requires ImageMagick (`convert`)
- Web UI accessible from LAN (binds to `:8080`, not just localhost)

## Non-goals (v1)
- Windows support
- OAuth (App Password is sufficient)
- Encryption beyond Gmail TLS
- Mobile app
