// Package server provides an HTTP process scaffold, without application routes.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/config"
)

func New(cfg config.Config) *http.Server {
	engine := gin.New()
	// No reverse proxy is configured for the local single-user server.
	_ = engine.SetTrustedProxies(nil)
	return &http.Server{
		Addr:              cfg.Address(),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Serve owns listener and drains active requests on context cancellation.
func Serve(ctx context.Context, srv *http.Server, listener net.Listener) error {
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()

	select {
	case err := <-done:
		return serveError(err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			<-done
			return fmt.Errorf("shutdown: %w", err)
		}
		return serveError(<-done)
	}
}

func serveError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
