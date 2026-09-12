// Пакет db: підключення до PostgreSQL (вбудований або зовнішній) і міграції.
package db

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/migrations"
)

// StartEmbedded піднімає вбудований PostgreSQL у dataDir/pg.
// Перший запуск завантажує бінарники (~40МБ), далі — стартує миттєво.
// Повертає рядок підключення і функцію зупинки.
func StartEmbedded(dataDir string, port uint) (string, func(), error) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V18).
			Port(uint32(port)).
			DataPath(filepath.Join(dataDir, "pg")).
			Username("jmw").
			Password("jmw").
			Database("jmw"),
	)
	if err := pg.Start(); err != nil {
		return "", nil, fmt.Errorf("embedded postgres: %w", err)
	}
	url := fmt.Sprintf("postgres://jmw:jmw@localhost:%d/jmw", port)
	return url, func() { _ = pg.Stop() }, nil
}

// Connect створює пул з'єднань і перевіряє доступність БД.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate застосовує вбудовані *.sql у порядку імен файлів, кожну в своїй транзакції.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
		   name       TEXT PRIMARY KEY,
		   applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		 )`); err != nil {
		return err
	}
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, name,
		).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sqlText, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sqlText)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
