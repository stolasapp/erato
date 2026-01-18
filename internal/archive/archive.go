// Package archive implements the ConnectRPC Archive service.
//
// The service uses a decorator (middleware) pattern where each layer wraps the
// previous one, adding specific functionality. Request flow is outside-in;
// response flow is inside-out.
//
// # Decorator Chain
//
// The chain is constructed innermost-first in [Default]:
//
//	Request → Validator → Paginator → Users → Interactivity → Hydrator → Scraper
//	                                                                        ↓
//	Response ← Validator ← Paginator ← Users ← Interactivity ← Hydrator ← Scraper
//
// Each decorator's role:
//
//   - Scraper: Fetches and parses content from the upstream archive
//   - Hydrator: Enriches resources with user-specific data (read times, bookmarks)
//   - Interactivity: Handles resource update operations (star, hide, mark read)
//   - Users: Implements user CRUD operations
//   - Paginator: Applies pagination and CEL filtering to list responses
//   - Validator: Validates requests before processing and responses after
//
// # Why Order Matters
//
// The Validator must be outermost to reject invalid requests before any
// processing occurs and to validate responses before they reach clients.
// The Paginator must wrap the data-providing decorators so it can filter and
// paginate their results. The Hydrator must run after Scraper so it can enrich
// the scraped resources with user data.
package archive

import (
	"log/slog"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/storage"
)

// Default returns a fully configured handler with the standard decorator chain.
// See package documentation for the chain order and rationale.
func Default(
	cfg *eratov1.Config,
	logger *slog.Logger,
	store storage.Store,
) (
	handler eratov1connect.ArchiveServiceHandler,
	err error,
) {
	handler, err = NewScraper(cfg, logger, eratov1connect.UnimplementedArchiveServiceHandler{})
	if err != nil {
		return nil, err
	}
	handler = NewHydrator(handler, store)
	handler = NewInteractivity(handler, store)
	handler = NewUsers(handler, store)
	if handler, err = NewPaginator(handler); err != nil {
		return nil, err
	}
	if handler, err = NewValidator(handler, logger); err != nil {
		return nil, err
	}
	return handler, nil
}
