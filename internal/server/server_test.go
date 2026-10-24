package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/zl14229/pokemon-shiny-tracker/internal/config"
)

func listen(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func awaitStop(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
		return nil
	}
}

func TestScaffoldServesAndStops(t *testing.T) {
	listener := listen(t)
	srv := New(config.Config{Port: 8080})
	t.Cleanup(func() { _ = srv.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, srv, listener) }()
	client := &http.Client{Timeout: time.Second}
	for _, path := range []string{"/", "/api/v1/health"} {
		resp, err := client.Get("http://" + listener.Addr().String() + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s status = %d; T001 must not expose application routes", path, resp.StatusCode)
		}
	}
	client.CloseIdleConnections()
	cancel()
	if err := awaitStop(t, done); err != nil {
		t.Fatal(err)
	}
	// A stopped server must release its listener so the local app can restart.
	reopened, err := net.Listen("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = reopened.Close()
}

func TestServeReportsListenerFailure(t *testing.T) {
	listener := listen(t)
	_ = listener.Close()
	if err := Serve(context.Background(), New(config.Config{Port: 8080}), listener); err == nil {
		t.Fatal("closed listener did not produce an error")
	}
}

func TestShutdownDrainsActiveRequest(t *testing.T) {
	listener := listen(t)
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	srv := New(config.Config{Port: 8080})
	// Test-only handler: no application route is registered in the scaffold.
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, "finished")
	})
	t.Cleanup(func() { _ = srv.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, srv, listener) }()
	response := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		defer client.CloseIdleConnections()
		resp, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			defer resp.Body.Close()
			var body []byte
			body, err = io.ReadAll(resp.Body)
			if err == nil && string(body) != "finished" {
				err = io.ErrUnexpectedEOF
			}
		}
		response <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	// RegisterOnShutdown proves cancellation reached Shutdown before release.
	shuttingDown := make(chan struct{})
	srv.RegisterOnShutdown(func() { close(shuttingDown) })
	cancel()
	select {
	case <-shuttingDown:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not start")
	}
	select {
	case err := <-done:
		t.Fatalf("shutdown returned before active request finished: %v", err)
	default:
	}
	release <- struct{}{}
	if err := awaitStop(t, response); err != nil {
		t.Fatalf("active request failed: %v", err)
	}
	if err := awaitStop(t, done); err != nil {
		t.Fatal(err)
	}
}
