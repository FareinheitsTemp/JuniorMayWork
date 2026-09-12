// JuniorMayWork — точка входу бекенду: API + WebSocket + скрейпер.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/api"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/archive"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/config"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/db"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/scraper"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/ws"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("база даних недоступна", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("міграції", "err", err)
		os.Exit(1)
	}
	log.Info("міграції застосовано")

	// origin-патерн для WS: з http://localhost:3000 → localhost:3000
	wsOrigins := []string{"localhost:3000", "127.0.0.1:3000"}
	if u, err := url.Parse(cfg.AllowedOrigin); err == nil && u.Host != "" {
		wsOrigins = []string{u.Host}
	}

	hub := ws.NewHub(log, wsOrigins)
	st := store.New(pool)
	ar := archive.New(cfg.ArchiveDir)
	manager := scraper.NewManager(st, ar, hub, cfg, log)
	server := api.NewServer(st, manager, hub, log)

	go manager.Run(ctx)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Routes(cfg.AllowedOrigin, hub.Handler()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("HTTP+WS слухає", "addr", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("завершення: даю серверу до 5с закритися")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}
