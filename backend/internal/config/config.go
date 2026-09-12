// Пакет config: конфігурація бекенду зі змінних оточення з дефолтами.
package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL   string
	Addr          string
	AllowedOrigin string
	PollInterval  time.Duration
	Sources       []string
	ArchiveDir    string
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	interval, err := time.ParseDuration(get("JMW_POLL_INTERVAL", "90s"))
	if err != nil || interval <= 0 {
		interval = 90 * time.Second
	}
	srcs := strings.Split(get("JMW_SOURCES", "upwork,reddit,weblancer"), ",")
	sources := make([]string, 0, len(srcs))
	for _, s := range srcs {
		if s = strings.TrimSpace(strings.ToLower(s)); s != "" {
			sources = append(sources, s)
		}
	}
	return Config{
		DatabaseURL:   get("JMW_DATABASE_URL", "postgres://jmw:jmw@localhost:5432/jmw"),
		Addr:          get("JMW_ADDR", ":8080"),
		AllowedOrigin: get("JMW_ALLOWED_ORIGIN", "http://localhost:3000"),
		PollInterval:  interval,
		Sources:       sources,
		ArchiveDir:    get("JMW_ARCHIVE_DIR", "./data/archive"),
	}
}
