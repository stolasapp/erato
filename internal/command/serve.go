package command

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/stolasapp/erato/internal/app"
	"github.com/stolasapp/erato/internal/app/devservice"
	"github.com/stolasapp/erato/internal/archive"
	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/sec"
	"github.com/stolasapp/erato/internal/server"
	"github.com/stolasapp/erato/internal/storage"
)

func serveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "serve the archive proxy and tracking tool RPC Server and Web App",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (runErr error) {
			cfg, logger, store, err := loadConfig(cmd.Context())
			if err != nil {
				return err
			}
			defer func() {
				if err := store.Close(); err != nil {
					runErr = errors.Join(runErr, err)
				}
			}()

			grp, ctx := errgroup.WithContext(cmd.Context())

			// In dev mode, start the fake upstream service
			if cfg.GetDevMode() {
				devAddr, err := serveDevUpstream(ctx, grp, logger)
				if err != nil {
					return err
				}
				cfg.SetRootUri("http://" + devAddr + "/")
			}

			rpcHandler, err := archive.Default(cfg, logger, store)
			if err != nil {
				return err
			}

			appServer := app.New(cfg, logger, store, rpcHandler)

			serveRPC(ctx, grp, cfg, logger, store, rpcHandler)
			serveApp(ctx, grp, cfg, logger, appServer)
			return grp.Wait()
		},
	}
}

func serveRPC(
	ctx context.Context,
	grp *errgroup.Group,
	cfg *eratov1.Config,
	logger *slog.Logger,
	store storage.Users,
	handler eratov1connect.ArchiveServiceHandler,
) {
	addr := cfg.GetRpcAddress()
	if addr == "" {
		return
	}

	listener, err := server.Listen(ctx, addr)
	if err != nil {
		grp.Go(func() error { return err })
		return
	}

	mux := http.NewServeMux()
	mux.Handle(eratov1connect.NewArchiveServiceHandler(handler))
	srv := &http.Server{Handler: sec.NewConnectAuthMiddleware(store).Wrap(mux)} //nolint:gosec // Serve() sets timeouts

	logger.InfoContext(ctx,
		"starting RPC server...",
		slog.String("address", addr),
	)
	server.Serve(ctx, grp, srv, listener, server.ShutdownTimeout)
}

func serveApp(
	ctx context.Context,
	grp *errgroup.Group,
	cfg *eratov1.Config,
	logger *slog.Logger,
	srv *echo.Echo,
) {
	addr := cfg.GetWebAddress()
	if addr == "" {
		return
	}

	listener, err := server.Listen(ctx, addr)
	if err != nil {
		grp.Go(func() error { return err })
		return
	}

	logger.InfoContext(ctx,
		"starting app server...",
		slog.String("address", addr),
	)
	server.Serve(ctx, grp, srv.Server, listener, server.ShutdownTimeout)
}

func serveDevUpstream(
	ctx context.Context,
	grp *errgroup.Group,
	logger *slog.Logger,
) (string, error) {
	seed := devservice.Seed()
	handler := devservice.New(seed)

	listener, err := server.Listen(ctx, "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()

	srv := &http.Server{Handler: handler} //nolint:gosec // Serve() sets timeouts

	logger.InfoContext(ctx,
		"starting dev upstream server...",
		slog.String("address", addr),
		slog.Uint64("seed", seed),
	)
	server.Serve(ctx, grp, srv, listener, server.ShutdownTimeout)

	return addr, nil
}
