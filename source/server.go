package source

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"
)

func StartServer(d *Daemon, addr string, webFS fs.FS) *http.Server {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.FS(webFS)))

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		running, last := d.Status()
		writeJSON(w, map[string]interface{}{
			"running":  running,
			"last_fax": formatTime(last),
		})
	})

	mux.HandleFunc("/api/log", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, d.Entries())
	})

	mux.HandleFunc("/api/debug", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, d.DebugLines())
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			cfg := d.CurrentConfig()
			cfg.Password = "" // never expose
			writeJSON(w, cfg)
		case "POST":
			var incoming Config
			if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			cfg := d.CurrentConfig()
			if incoming.Email != "" {
				cfg.Email = incoming.Email
			}
			if incoming.Password != "" {
				cfg.Password = incoming.Password
			}
			if incoming.PollSeconds >= 5 {
				cfg.PollSeconds = incoming.PollSeconds
			}
			if incoming.MaxMB > 0 {
				cfg.MaxMB = incoming.MaxMB
			}
			if incoming.Extensions != nil {
				cfg.Extensions = incoming.Extensions
			}
			if incoming.Senders != nil {
				cfg.Senders = incoming.Senders
			}
			cfg.Monochrome = incoming.Monochrome
			cfg.Scaling = incoming.Scaling
			d.UpdateConfig(cfg)
			writeJSON(w, map[string]string{"status": "ok"})
		default:
			http.Error(w, "method not allowed", 405)
		}
	})

	mux.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		to := r.FormValue("to")
		if to == "" || !strings.Contains(to, "@") {
			http.Error(w, "valid 'to' address required", 400)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file required", 400)
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		cfg := d.CurrentConfig()
		if !extAllowed(ext, cfg.Extensions) {
			http.Error(w, "file type not allowed", 400)
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "read error", 500)
			return
		}

		if err := sendFaxEmail(cfg, to, header.Filename, data); err != nil {
			log.Printf("send error: %v", err)
			http.Error(w, "send failed: "+err.Error(), 500)
			return
		}

		writeJSON(w, map[string]string{"status": "sent"})
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("web UI at http://%s", addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()
	return srv
}

func sendFaxEmail(cfg Config, to, filename string, data []byte) error {
	boundary := fmt.Sprintf("faxd-%d", time.Now().UnixNano())
	ct := mime.TypeByExtension(filepath.Ext(filename))
	if ct == "" {
		ct = "application/octet-stream"
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("From: %s\r\n", cfg.Email))
	body.WriteString(fmt.Sprintf("To: %s\r\n", to))
	body.WriteString(fmt.Sprintf("Subject: Fax: %s\r\n", filename))
	body.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
	body.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))

	body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	body.WriteString("Content-Type: text/plain\r\n\r\n")
	body.WriteString("Fax sent via faxd.\r\n\r\n")

	body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	body.WriteString(fmt.Sprintf("Content-Type: %s\r\n", ct))
	body.WriteString("Content-Transfer-Encoding: base64\r\n")
	body.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n\r\n", filename))
	body.WriteString(base64.StdEncoding.EncodeToString(data))
	body.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	auth := smtp.PlainAuth("", cfg.Email, cfg.Password, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, cfg.Email, []string{to}, []byte(body.String()))
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format(time.RFC3339)
}
