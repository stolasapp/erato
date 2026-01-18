// Package server provides shared HTTP server utilities.
package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

// Server timeouts.
const (
	ReadHeaderTimeout = 1 * time.Second
	ReadTimeout       = 5 * time.Second
	WriteTimeout      = 5 * time.Second
	ShutdownTimeout   = 10 * time.Second
)

// Listen creates a TCP listener on the given address.
// Use "127.0.0.1:0" for a random available port.
func Listen(ctx context.Context, addr string) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(ctx, "tcp", addr)
}

// Serve starts an HTTP server on the given listener and registers graceful
// shutdown when the context is canceled. The server is configured with
// standard timeouts.
func Serve(
	ctx context.Context,
	grp *errgroup.Group,
	srv *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
) {
	srv.ReadHeaderTimeout = ReadHeaderTimeout
	srv.ReadTimeout = ReadTimeout
	srv.WriteTimeout = WriteTimeout

	grp.Go(func() error {
		err := srv.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	grp.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})
}
