// Пакет config: конфігурація бекенду зі змінних оточення з дефолтами.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// DatabaseURL: порожня => база вбудована (embedded-postgres).
	// Встанови JMW_DATABASE_URL, щоб ходити у зовнішній PostgreSQL.
	DatabaseURL   string
	EmbeddedPort  uint
	DataDir       string
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
	port, err := strconv.ParseUint(get("JMW_EMBEDDED_PORT", "5433"), 10, 16)
	if err != nil {
		port = 5433
	}
	srcs := strings.Split(get("JMW_SOURCES", "telegram,freelancehunt,weblancer"), ",")
	sources := make([]string, 0, len(srcs))
	for _, s := range srcs {
		if s = strings.TrimSpace(strings.ToLower(s)); s != "" {
			sources = append(sources, s)
		}
	}
	return Config{
		DatabaseURL:   get("JMW_DATABASE_URL", ""),
		EmbeddedPort:  uint(port),
		DataDir:       get("JMW_DATA_DIR", "./data"),
		Addr:          get("JMW_ADDR", ":8080"),
		AllowedOrigin: get("JMW_ALLOWED_ORIGIN", "http://localhost:3000"),
		PollInterval:  interval,
		Sources:       sources,
		ArchiveDir:    get("JMW_ARCHIVE_DIR", "./data/archive"),
	}
}
