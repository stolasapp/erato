// Package uitest provides UI testing utilities using Rod.
package uitest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	"github.com/stolasapp/erato/internal/app"
	"github.com/stolasapp/erato/internal/app/devservice"
	"github.com/stolasapp/erato/internal/archive"
	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/server"
	"github.com/stolasapp/erato/internal/slugconv"
	"github.com/stolasapp/erato/internal/storage"
	"github.com/stolasapp/erato/internal/storage/db"
)

// TestSeed is the fixed seed used for reproducible test data.
const TestSeed uint64 = 12345

// Server is a test server that runs the app in dev mode.
type Server struct {
	baseURL string
	cancel  context.CancelFunc
	grp     *errgroup.Group
	store   storage.Store
}

// newTestServer creates and starts a new test server for use in TestMain.
// It panics on errors since TestMain cannot use testing.TB.
func newTestServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	grp, ctx := errgroup.WithContext(ctx)

	logger := slog.New(slog.DiscardHandler)

	// Create in-memory storage
	cfg := testConfig()
	store, err := storage.NewDB(ctx, cfg, logger)
	if err != nil {
		cancel()
		panic(fmt.Sprintf("failed to create storage: %v", err))
	}

	// Start dev upstream service
	devAddr, err := startDevUpstream(ctx, grp)
	if err != nil {
		cancel()
		_ = store.Close()
		panic(fmt.Sprintf("failed to start dev upstream: %v", err))
	}
	cfg.SetRootUri("http://" + devAddr + "/")

	// Create archive handler
	rpcHandler, err := archive.Default(cfg, logger, store)
	if err != nil {
		cancel()
		_ = store.Close()
		panic(fmt.Sprintf("failed to create archive handler: %v", err))
	}

	// Create and start app server
	appServer := app.New(cfg, logger, store, rpcHandler)
	appAddr, err := startAppServer(ctx, grp, appServer)
	if err != nil {
		cancel()
		_ = store.Close()
		panic(fmt.Sprintf("failed to start app server: %v", err))
	}

	return &Server{
		baseURL: "http://" + appAddr,
		cancel:  cancel,
		grp:     grp,
		store:   store,
	}
}

// BaseURL returns the base URL of the test server.
func (s *Server) BaseURL() string {
	return s.baseURL
}

// Close shuts down the test server.
// Errors are ignored since this runs during test cleanup where failures
// are typically unrecoverable and already logged by the errgroup.
func (s *Server) Close() {
	s.cancel()
	_ = s.grp.Wait()
	_ = s.store.Close()
}

func testConfig() *eratov1.Config {
	return eratov1.Config_builder{
		LogLevel:   eratov1.Config_DEBUG,
		DbFilepath: ":memory:",
		DevMode:    true,
	}.Build()
}

func startDevUpstream(ctx context.Context, grp *errgroup.Group) (string, error) {
	handler := devservice.New(TestSeed)

	listener, err := server.Listen(ctx, "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()

	srv := &http.Server{Handler: handler} //nolint:gosec // Serve() sets timeouts
	server.Serve(ctx, grp, srv, listener, server.ShutdownTimeout)

	return addr, nil
}

func startAppServer(ctx context.Context, grp *errgroup.Group, srv *echo.Echo) (string, error) {
	listener, err := server.Listen(ctx, "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()

	server.Serve(ctx, grp, srv.Server, listener, server.ShutdownTimeout)

	return addr, nil
}

// URL constructs a full URL from the server base URL and a path.
func (s *Server) URL(path string) string {
	return fmt.Sprintf("%s%s", s.baseURL, path)
}

// SetViewTime sets the view_time for a resource to a specific time.
// The slug is the DOM element ID (e.g., "adventure/sheaf-and-horse").
// This is useful for testing the "updated" timestamp state where
// update_time > view_time.
func (s *Server) SetViewTime(ctx context.Context, slug string, viewTime time.Time) error {
	path, err := slugToPath(slug)
	if err != nil {
		return err
	}
	return s.store.UpsertResource(ctx, db.Resource{
		User: 0, // Dev mode skips auth, so user ID is 0
		Path: path,
		ViewTime: sql.NullTime{
			Time:  viewTime,
			Valid: true,
		},
	})
}

// Slug segment counts for different resource types.
const (
	categorySegments = 1
	entrySegments    = 2
	chapterSegments  = 3
)

// slugToPath converts a slug (DOM ID) to a resource path.
// It determines the resource type by counting path segments.
func slugToPath(slug string) (string, error) {
	segments := strings.Count(slug, "/") + 1
	switch segments {
	case categorySegments:
		return slugconv.ToCategoryPath(slug)
	case entrySegments:
		return slugconv.ToEntryPath(slug)
	case chapterSegments:
		return slugconv.ToChapterPath(slug)
	default:
		return "", fmt.Errorf("invalid slug: %q", slug)
	}
}
