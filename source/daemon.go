package source

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FaxEntry struct {
	Sender   string    `json:"sender"`
	Time     time.Time `json:"time"`
	Filename string    `json:"filename"`
	Status   string    `json:"status"`
}

const maxDebugLines = 200

type Daemon struct {
	mu        sync.Mutex
	cfg       Config
	log       []FaxEntry
	debug     []string
	running   bool
	lastFax   time.Time
	lastCheck time.Time
	stop      chan struct{}
}

func NewDaemon(cfg Config) *Daemon {
	d := &Daemon{
		cfg:  cfg,
		stop: make(chan struct{}),
	}
	log.SetOutput(&logWriter{d})
	log.SetFlags(log.Ltime)
	d.loadLog()
	d.loadState()
	return d
}

func (d *Daemon) Start() {
	d.mu.Lock()
	d.running = true
	d.mu.Unlock()

	ticker := time.NewTicker(time.Duration(d.cfg.PollSeconds) * time.Second)
	defer ticker.Stop()

	d.poll()

	for {
		select {
		case <-ticker.C:
			d.poll()
		case <-d.stop:
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
			return
		}
	}
}

func (d *Daemon) Shutdown() {
	close(d.stop)
}

func (d *Daemon) poll() {
	d.mu.Lock()
	cfg := d.cfg
	since := d.lastCheck
	d.mu.Unlock()

	entries, err := fetchNewFaxes(cfg, since)
	if err != nil {
		log.Printf("poll error: %v", err)
		return
	}

	now := time.Now()
	d.mu.Lock()
	d.lastCheck = now
	if len(entries) > 0 {
		d.log = append(d.log, entries...)
		d.lastFax = entries[len(entries)-1].Time
	}
	d.mu.Unlock()

	d.saveState()
	if len(entries) > 0 {
		d.saveLog()
		log.Printf("received %d fax(es)", len(entries))
	}
}

func (d *Daemon) Status() (bool, time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running, d.lastFax
}

func (d *Daemon) Entries() []FaxEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]FaxEntry, len(d.log))
	copy(out, d.log)
	return out
}

func (d *Daemon) UpdateConfig(cfg Config) {
	d.mu.Lock()
	d.cfg = cfg
	d.mu.Unlock()
	saveConfig(cfg)
}

func (d *Daemon) CurrentConfig() Config {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cfg
}

// logWriter tees log output to stderr and a ring buffer.
type logWriter struct {
	d *Daemon
}

func (w *logWriter) Write(p []byte) (int, error) {
	os.Stderr.Write(p)
	line := strings.TrimRight(string(p), "\n")
	w.d.mu.Lock()
	w.d.debug = append(w.d.debug, line)
	if len(w.d.debug) > maxDebugLines {
		w.d.debug = w.d.debug[len(w.d.debug)-maxDebugLines:]
	}
	w.d.mu.Unlock()
	return len(p), nil
}

func (d *Daemon) DebugLines() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.debug))
	copy(out, d.debug)
	return out
}

func logPath() string {
	return filepath.Join(dataDir(), "log.json")
}

func (d *Daemon) loadLog() {
	data, err := os.ReadFile(logPath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &d.log)
	if len(d.log) > 0 {
		d.lastFax = d.log[len(d.log)-1].Time
	}
}

func (d *Daemon) saveLog() {
	d.mu.Lock()
	data, _ := json.MarshalIndent(d.log, "", "  ")
	d.mu.Unlock()
	os.MkdirAll(dataDir(), 0755)
	os.WriteFile(logPath(), data, 0644)
}

type daemonState struct {
	LastCheck time.Time `json:"last_check"`
}

func statePath() string {
	return filepath.Join(dataDir(), "state.json")
}

func (d *Daemon) loadState() {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return
	}
	var s daemonState
	if json.Unmarshal(data, &s) == nil {
		d.lastCheck = s.LastCheck
	}
}

func (d *Daemon) saveState() {
	d.mu.Lock()
	s := daemonState{LastCheck: d.lastCheck}
	d.mu.Unlock()
	data, _ := json.Marshal(s)
	os.MkdirAll(dataDir(), 0755)
	os.WriteFile(statePath(), data, 0644)
}
