package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stolasapp/erato/internal/app/component"
	"github.com/stolasapp/erato/internal/app/component/page"
	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/slugconv"
)

type handler struct {
	handler eratov1connect.ArchiveServiceHandler
}

func (h handler) register(e *echo.Echo) {
	e.GET("/", h.archive)

	category := e.Group("/:category")
	category.GET("", h.category)
	category.PUT("/ops/:op", h.categoryOp)

	entry := category.Group("/:entry")
	entry.GET("", h.entry)
	entry.PUT("/ops/:op", h.entryOp)

	chapter := entry.Group("/:chapter")
	chapter.GET("", h.chapter)
	chapter.PUT("/ops/:op", h.chapterOp)
}

func (h handler) archive(c echo.Context) error {
	resp, err := h.handler.ListCategories(
		c.Request().Context(),
		connect.NewRequest(eratov1.ListCategoriesRequest_builder{}.Build()),
	)
	if err != nil {
		return err
	}

	filters := parseFilterParams(c)

	// HTMX request - return just the list component
	if isHTMX(c) {
		return component.CategoryList(resp.Msg.GetResults(), component.ListProps{
			Title:    "Categories",
			ListType: component.ListTypeCategories,
			Filters:  filters,
			BaseURL:  "/",
		}).Render(
			c.Request().Context(),
			c.Response().Writer,
		)
	}

	return render(
		c.Request().Context(),
		page.Archive(resp.Msg.GetResults(), filters),
		c.Response().Writer,
	)
}

func (h handler) category(c echo.Context) error {
	slug := c.Param("category")
	path, err := slugconv.ToCategoryPath(slug)
	if err != nil {
		return err
	}

	category, err := h.handler.GetCategory(
		c.Request().Context(),
		connect.NewRequest(eratov1.GetCategoryRequest_builder{Path: path}.Build()),
	)
	if err != nil {
		return fmt.Errorf("failed to get category %q: %w", path, err)
	}

	filters := parseFilterParams(c)
	entries, err := h.handler.ListEntries(
		c.Request().Context(),
		connect.NewRequest(eratov1.ListEntriesRequest_builder{
			Parent:      path,
			Filter:      buildEntryFilter(filters.Filters),
			MaxPageSize: defaultPageSize,
			PageToken:   filters.Page,
		}.Build()),
	)
	if err != nil {
		return err
	}

	// Hidden filtering is done via CSS to preserve fragment navigation
	listProps := component.ListProps{
		Title:         category.Msg.GetDisplayName(),
		ListType:      component.ListTypeMixed,
		Filters:       filters,
		BaseURL:       "/" + slug,
		NextPageToken: entries.Msg.GetNextPageToken(),
	}

	// HTMX request - return just the list component
	if isHTMX(c) {
		return component.EntryList(entries.Msg.GetResults(), listProps).Render(
			c.Request().Context(),
			c.Response().Writer,
		)
	}

	return page.Category(
		category.Msg,
		entries.Msg.GetResults(),
		listProps,
	).Render(
		c.Request().Context(),
		c.Response().Writer,
	)
}

func (h handler) entry(c echo.Context) error {
	slug := c.Param("category") + "/" + c.Param("entry")
	path, err := slugconv.ToEntryPath(slug)
	if err != nil {
		return err
	}

	entry, err := h.handler.GetEntry(
		c.Request().Context(),
		connect.NewRequest(eratov1.GetEntryRequest_builder{
			Path: path,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	if entry.Msg.GetKind() == eratov1.Entry_ANTHOLOGY {
		return h.anthology(c, entry.Msg)
	}
	return h.story(c, entry.Msg)
}

func (h handler) story(c echo.Context, entry *eratov1.Entry) error {
	content, err := h.handler.ReadEntry(
		c.Request().Context(),
		connect.NewRequest(eratov1.ReadEntryRequest_builder{
			Path:     entry.GetPath(),
			MimeType: eratov1.ReadEntryRequest_HTML,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	filters := parseFilterParams(c)
	err = page.Story(
		entry,
		content.Msg.GetContent(),
		filters,
	).Render(
		c.Request().Context(),
		c.Response().Writer,
	)
	if err != nil {
		return toHTTPError(err)
	}

	h.markEntryViewed(c.Request().Context(), entry.GetPath())
	return nil
}

func (h handler) anthology(c echo.Context, entry *eratov1.Entry) error {
	slug := c.Param("category") + "/" + c.Param("entry")
	filters := parseFilterParams(c)
	chapters, err := h.handler.ListChapters(
		c.Request().Context(),
		connect.NewRequest(eratov1.ListChaptersRequest_builder{
			Parent:      entry.GetPath(),
			MaxPageSize: defaultPageSize,
			PageToken:   filters.Page,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	listProps := component.ListProps{
		Title:         entry.GetDisplayName(),
		ListType:      component.ListTypeChapters,
		Filters:       filters,
		BaseURL:       "/" + slug,
		NextPageToken: chapters.Msg.GetNextPageToken(),
	}

	err = page.Anthology(
		entry,
		chapters.Msg.GetResults(),
		listProps,
	).Render(
		c.Request().Context(),
		c.Response().Writer,
	)
	if err != nil {
		return toHTTPError(err)
	}

	h.markEntryViewed(c.Request().Context(), entry.GetPath())
	return nil
}

func (h handler) chapter(c echo.Context) error {
	slug := c.Param("category") + "/" + c.Param("entry") + "/" + c.Param("chapter")
	path, err := slugconv.ToChapterPath(slug)
	if err != nil {
		return err
	}

	chapter, err := h.handler.GetChapter(
		c.Request().Context(),
		connect.NewRequest(eratov1.GetChapterRequest_builder{
			Path: path,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	content, err := h.handler.ReadChapter(
		c.Request().Context(),
		connect.NewRequest(eratov1.ReadChapterRequest_builder{
			Path:     chapter.Msg.GetPath(),
			MimeType: eratov1.ReadEntryRequest_HTML,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	filters := parseFilterParams(c)
	err = page.Chapter(
		chapter.Msg,
		content.Msg.GetContent(),
		filters,
	).Render(
		c.Request().Context(),
		c.Response().Writer,
	)
	if err != nil {
		return toHTTPError(err)
	}

	h.markChapterViewed(c.Request().Context(), chapter.Msg.GetPath())
	return nil
}

func (h handler) entryOp(c echo.Context) error {
	slug := c.Param("category") + "/" + c.Param("entry")
	path, err := slugconv.ToEntryPath(slug)
	if err != nil {
		return err
	}

	op := c.Param("op")
	entry, mask := applyEntryOp(op)
	if mask == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid operation")
	}

	_, err = h.handler.UpdateEntry(
		c.Request().Context(),
		connect.NewRequest(eratov1.UpdateEntryRequest_builder{
			Path:       path,
			Entry:      entry,
			UpdateMask: mask,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	// Re-fetch the entry to get the complete state
	updated, err := h.handler.GetEntry(
		c.Request().Context(),
		connect.NewRequest(eratov1.GetEntryRequest_builder{
			Path: path,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	return renderOp(c, updated.Msg, component.EntryContentActions, func(e *eratov1.Entry) templ.Component {
		return component.EntryItem(e, component.FilterParams{})
	})
}

func applyEntryOp(op string) (*eratov1.Entry, *fieldmaskpb.FieldMask) {
	entry := &eratov1.Entry{}
	switch op {
	case "view":
		entry.SetViewTime(timestamppb.Now())
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"view_time"}}
	case "unview":
		entry.ClearViewTime()
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"view_time"}}
	case "read":
		entry.SetReadTime(timestamppb.Now())
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"read_time"}}
	case "unread":
		entry.ClearReadTime()
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"read_time"}}
	case "star":
		entry.SetStarred(true)
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"starred"}}
	case "unstar":
		entry.SetStarred(false)
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"starred"}}
	case "hide":
		entry.SetHidden(true)
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"hidden"}}
	case "unhide":
		entry.SetHidden(false)
		return entry, &fieldmaskpb.FieldMask{Paths: []string{"hidden"}}
	default:
		return nil, nil
	}
}

func (h handler) chapterOp(c echo.Context) error {
	slug := c.Param("category") + "/" + c.Param("entry") + "/" + c.Param("chapter")
	path, err := slugconv.ToChapterPath(slug)
	if err != nil {
		return err
	}

	op := c.Param("op")
	chapter, mask := applyChapterOp(op)
	if mask == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid operation")
	}

	_, err = h.handler.UpdateChapter(
		c.Request().Context(),
		connect.NewRequest(eratov1.UpdateChapterRequest_builder{
			Path:       path,
			Chapter:    chapter,
			UpdateMask: mask,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	// Re-fetch the chapter to get the complete state
	updated, err := h.handler.GetChapter(
		c.Request().Context(),
		connect.NewRequest(eratov1.GetChapterRequest_builder{
			Path: path,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	return renderOp(c, updated.Msg, component.ChapterContentActions, func(ch *eratov1.Chapter) templ.Component {
		return component.ChapterItem(ch, component.FilterParams{})
	})
}

func (h handler) categoryOp(c echo.Context) error {
	slug := c.Param("category")
	path, err := slugconv.ToCategoryPath(slug)
	if err != nil {
		return err
	}

	op := c.Param("op")
	category, mask := applyCategoryOp(op)
	if mask == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid operation")
	}

	_, err = h.handler.UpdateCategory(
		c.Request().Context(),
		connect.NewRequest(eratov1.UpdateCategoryRequest_builder{
			Path:       path,
			Category:   category,
			UpdateMask: mask,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	// Re-fetch the category to get the complete state
	updated, err := h.handler.GetCategory(
		c.Request().Context(),
		connect.NewRequest(eratov1.GetCategoryRequest_builder{
			Path: path,
		}.Build()),
	)
	if err != nil {
		return toHTTPError(err)
	}

	return component.CategoryItem(updated.Msg, component.FilterParams{}).Render(
		c.Request().Context(),
		c.Response().Writer,
	)
}

func applyChapterOp(op string) (*eratov1.Chapter, *fieldmaskpb.FieldMask) {
	chapter := &eratov1.Chapter{}
	switch op {
	case "view":
		chapter.SetViewTime(timestamppb.Now())
		return chapter, &fieldmaskpb.FieldMask{Paths: []string{"view_time"}}
	case "unview":
		chapter.ClearViewTime()
		return chapter, &fieldmaskpb.FieldMask{Paths: []string{"view_time"}}
	case "read":
		chapter.SetReadTime(timestamppb.Now())
		return chapter, &fieldmaskpb.FieldMask{Paths: []string{"read_time"}}
	case "unread":
		chapter.ClearReadTime()
		return chapter, &fieldmaskpb.FieldMask{Paths: []string{"read_time"}}
	default:
		return nil, nil
	}
}

func applyCategoryOp(op string) (*eratov1.Category, *fieldmaskpb.FieldMask) {
	category := &eratov1.Category{}
	switch op {
	case "hide":
		category.SetHidden(true)
		return category, &fieldmaskpb.FieldMask{Paths: []string{"hidden"}}
	case "unhide":
		category.SetHidden(false)
		return category, &fieldmaskpb.FieldMask{Paths: []string{"hidden"}}
	default:
		return nil, nil
	}
}

func (h handler) markEntryViewed(ctx context.Context, path string) {
	entry, mask := applyEntryOp("view")
	_, err := h.handler.UpdateEntry(
		ctx,
		connect.NewRequest(eratov1.UpdateEntryRequest_builder{
			Path:       path,
			Entry:      entry,
			UpdateMask: mask,
		}.Build()),
	)
	if err != nil {
		slog.Error("failed to mark entry as viewed",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
}

func (h handler) markChapterViewed(ctx context.Context, path string) {
	chapter, mask := applyChapterOp("view")
	_, err := h.handler.UpdateChapter(
		ctx,
		connect.NewRequest(eratov1.UpdateChapterRequest_builder{
			Path:       path,
			Chapter:    chapter,
			UpdateMask: mask,
		}.Build()),
	)
	if err != nil {
		slog.Error("failed to mark chapter as viewed",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
}

const htmxTrue = "true"
const defaultPageSize = 100

func isHTMX(c echo.Context) bool {
	return c.Request().Header.Get("Hx-Request") == htmxTrue
}

func parseFilterParams(c echo.Context) component.FilterParams {
	return component.FilterParams{
		Filters: component.Filters{
			TypeFilter:  c.QueryParam("type"),
			OnlyUnread:  c.QueryParam("unread") == htmxTrue,
			OnlyStarred: c.QueryParam("starred") == htmxTrue,
			ShowHidden:  c.QueryParam("hidden") == htmxTrue,
		},
		Page:   c.QueryParam("page"),
		Parent: c.QueryParam("pf"),
	}
}

func buildEntryFilter(filters component.Filters) string {
	var conditions []string
	if filters.OnlyUnread {
		conditions = append(conditions, "!has(this.read_time)")
	}
	if filters.OnlyStarred {
		conditions = append(conditions, "this.starred")
	}
	if filters.TypeFilter == "story" {
		conditions = append(conditions, fmt.Sprintf("this.kind == %d", eratov1.Entry_STORY.Number()))
	}
	if filters.TypeFilter == "anthology" {
		conditions = append(conditions, fmt.Sprintf("this.kind == %d", eratov1.Entry_ANTHOLOGY.Number()))
	}
	return strings.Join(conditions, " && ")
}

// toHTTPError converts an error to an Echo HTTPError with the appropriate
// HTTP status code. ConnectRPC errors are mapped to their corresponding HTTP
// status codes; other errors pass through unchanged.
func toHTTPError(err error) error {
	if err == nil {
		return nil
	}

	// Already an HTTP error - pass through
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return err
	}

	// Map ConnectRPC codes to HTTP status codes
	code := connect.CodeOf(err)
	status := connectCodeToHTTPStatus(code)
	if status != http.StatusInternalServerError {
		return echo.NewHTTPError(status, err.Error())
	}

	// Unknown code or non-Connect error - return as-is for default handling
	return err
}

// connectCodeToHTTPStatus maps ConnectRPC error codes to HTTP status codes.
// See: https://connectrpc.com/docs/protocol/#error-codes
func connectCodeToHTTPStatus(code connect.Code) int {
	switch code {
	case connect.CodeInvalidArgument, connect.CodeFailedPrecondition, connect.CodeOutOfRange:
		return http.StatusBadRequest // 400
	case connect.CodeUnauthenticated:
		return http.StatusUnauthorized // 401
	case connect.CodePermissionDenied:
		return http.StatusForbidden // 403
	case connect.CodeNotFound:
		return http.StatusNotFound // 404
	case connect.CodeCanceled:
		return http.StatusRequestTimeout // 408
	case connect.CodeAlreadyExists, connect.CodeAborted:
		return http.StatusConflict // 409
	case connect.CodeResourceExhausted:
		return http.StatusTooManyRequests // 429
	case connect.CodeUnimplemented:
		return http.StatusNotImplemented // 501
	case connect.CodeUnavailable:
		return http.StatusServiceUnavailable // 503
	case connect.CodeDeadlineExceeded:
		return http.StatusGatewayTimeout // 504
	case connect.CodeInternal, connect.CodeDataLoss, connect.CodeUnknown:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError // 500
	}
}

var renderBufferPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func render(ctx context.Context, component templ.Component, w io.Writer) error {
	buf := renderBufferPool.Get().(*bytes.Buffer) //nolint:forcetypeassert // guaranteed by impl
	defer renderBufferPool.Put(buf)
	buf.Reset()

	if err := component.Render(ctx, buf); err != nil {
		return toHTTPError(err)
	}
	_, err := io.Copy(w, buf)
	return toHTTPError(err)
}

// renderOp renders either the content actions or list item component based on
// the Hx-Target header. If the target is "content-actions" (content page actions),
// it renders the content actions; otherwise, it renders the list item.
func renderOp[T any](c echo.Context, item T, contentActions, listItem func(T) templ.Component) error {
	target := c.Request().Header.Get("Hx-Target")
	comp := listItem(item)
	if target == "content-actions" {
		comp = contentActions(item)
	}
	return comp.Render(c.Request().Context(), c.Response().Writer)
}
