package source

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Email        string   `toml:"email"         json:"email"`
	Password     string   `toml:"password"      json:"password"`
	PollSeconds  int      `toml:"poll_interval_seconds" json:"poll_interval_seconds"`
	Senders      []string `toml:"allowed_senders"       json:"allowed_senders"`
	MaxMB        int      `toml:"max_attachment_mb"      json:"max_attachment_mb"`
	Extensions   []string `toml:"allowed_extensions"     json:"allowed_extensions"`
	Monochrome   bool     `toml:"monochrome"    json:"monochrome"`
	Scaling      int      `toml:"scaling"       json:"scaling"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "faxd")
}

func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

func dataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "faxd")
}

func receivedDir() string {
	return filepath.Join(dataDir(), "received")
}

func LoadConfig() (Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(configPath(), &cfg)
	if os.IsNotExist(err) {
		cfg = defaultConfig()
		saveConfig(cfg)
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if cfg.PollSeconds == 0 {
		cfg.PollSeconds = 30
	}
	if cfg.MaxMB == 0 {
		cfg.MaxMB = 20
	}
	if len(cfg.Extensions) == 0 {
		cfg.Extensions = []string{".pdf", ".jpg", ".jpeg", ".png"}
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		PollSeconds: 30,
		MaxMB:       20,
		Extensions:  []string{".pdf", ".jpg", ".jpeg", ".png"},
		Monochrome:  true,
		Scaling:     50,
	}
}

func saveConfig(cfg Config) error {
	os.MkdirAll(configDir(), 0755)
	f, err := os.Create(configPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
