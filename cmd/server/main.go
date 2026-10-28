package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/api"
	"github.com/zl14229/pokemon-shiny-tracker/internal/config"
	"github.com/zl14229/pokemon-shiny-tracker/internal/database"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"github.com/zl14229/pokemon-shiny-tracker/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	gin.SetMode(gin.ReleaseMode)
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	conn, err := db.DB()
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := reference.EnsureSeed(db); err != nil {
		return err
	}
	if err := reference.EnsureCatalog(db); err != nil {
		slog.Warn("catalog enrichment unavailable; local records remain available")
	}
	if err := huntdata.EnsureBundled(db); err != nil {
		slog.Warn("hunt reference unavailable; personal records remain available")
	}
	srv := server.New(cfg, api.NewRouter(db))
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("server listening", "address", listener.Addr().String())
	return server.Serve(ctx, srv, listener)
}
