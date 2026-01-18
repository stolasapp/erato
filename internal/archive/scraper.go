package archive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/PuerkitoBio/goquery"
	"github.com/die-net/lrucache"
	"github.com/gocolly/colly/v2"
	"github.com/gregjones/httpcache"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stolasapp/erato/internal/content"
	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/pagination"
	"github.com/stolasapp/erato/internal/slugconv"
)

const (
	userAgent         = "okhttp/4.9.2"
	maxHTTPCacheBytes = 256 * 1024 * 1024 // 256 MiB
	maxHTTPCacheAge   = 0                 // unlimited
	idleConns         = 100
	idleConnTimeout   = 90 * time.Second
	httpTimeout       = 10 * time.Second

	rowCSSSelector = "div.ftr,tr:not(:first-child)"
)

// Scraper is a [eratov1connect.ArchiveServiceHandler] that extracts archive
// data from the upstream root URI. This implementation only handles read paths
// for accessing archive entities; decorators must provide the write paths. This
// implementation also does not handle pagination.
type Scraper struct {
	eratov1connect.ArchiveServiceHandler

	base   *url.URL
	client *http.Client
	logger *slog.Logger
	locale *time.Location
}

// NewScraper creates a Scraper with the provided config and base logger.
func NewScraper(
	cfg *eratov1.Config,
	logger *slog.Logger,
	inner eratov1connect.ArchiveServiceHandler,
) (*Scraper, error) {
	base, err := url.Parse(cfg.GetRootUri())
	if err != nil {
		return nil, fmt.Errorf("failed to parse root uri: %w", err)
	} else if !base.IsAbs() {
		return nil, fmt.Errorf("root uri must have a scheme: %v", base)
	}

	const localeName = "America/New_York"
	locale, err := time.LoadLocation(localeName)
	if err != nil {
		return nil, fmt.Errorf("failed to load locale %q: %w", localeName, err)
	}

	return &Scraper{
		ArchiveServiceHandler: inner,
		base:                  base,
		client: &http.Client{
			Transport: &httpcache.Transport{
				Cache: lrucache.New(maxHTTPCacheBytes, maxHTTPCacheAge),
				Transport: &http.Transport{
					Proxy:               http.ProxyFromEnvironment,
					ForceAttemptHTTP2:   true,
					MaxIdleConns:        idleConns,
					MaxConnsPerHost:     idleConns,
					MaxIdleConnsPerHost: idleConns,
					IdleConnTimeout:     idleConnTimeout,
					TLSHandshakeTimeout: httpTimeout,
				},
			},
			Timeout: httpTimeout,
		},
		logger: logger.With(slog.String("component", "scraper")),
		locale: locale,
	}, nil
}

// ListCategories satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) ListCategories(
	ctx context.Context,
	_ *connect.Request[eratov1.ListCategoriesRequest],
) (res *connect.Response[eratov1.ListCategoriesResponse], retErr error) {
	bldr := eratov1.ListCategoriesResponse_builder{}

	col := s.newCollector(ctx)
	col.OnHTML("body", func(body *colly.HTMLElement) {
		ct := body.DOM.Find(".list-group-item").Length()
		bldr.Results = slices.Grow(bldr.Results, ct)
	})
	col.OnHTML(".list-group-item", func(elem *colly.HTMLElement) {
		slug := elem.ChildAttr("a", "href")
		slug = strings.TrimPrefix(slug, s.base.Path)
		categoryPath, err := slugconv.ToCategoryPath(slug)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to resolve category path",
				slog.String("slug", slug),
				slog.Any("error", err),
			)
			return
		}
		cat := eratov1.Category_builder{
			Path:        categoryPath,
			DisplayName: elem.ChildText("a"),
		}
		elem.DOM.Contents().EachWithBreak(func(_ int, sel *goquery.Selection) bool {
			if goquery.NodeName(sel) == "#text" {
				cat.Description = strings.TrimLeft(sel.Text(), " -")
				return false
			}
			return true
		})
		bldr.Results = append(bldr.Results, cat.Build())
	})

	if err := col.Visit(s.base.String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scrape categories from %v: %w", s.base, err))
	}
	return connect.NewResponse(bldr.Build()), nil
}

// GetCategory satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) GetCategory(
	ctx context.Context,
	req *connect.Request[eratov1.GetCategoryRequest],
) (*connect.Response[eratov1.Category], error) {
	res, err := s.ListCategories(ctx, connect.NewRequest(&eratov1.ListCategoriesRequest{}))
	if err != nil {
		return nil, err
	}
	for _, cat := range res.Msg.GetResults() {
		if cat.GetPath() == req.Msg.GetPath() {
			return connect.NewResponse(cat), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, nil)
}

// ListEntries satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) ListEntries(
	ctx context.Context,
	req *connect.Request[eratov1.ListEntriesRequest],
) (*connect.Response[eratov1.ListEntriesResponse], error) {
	bldr := eratov1.ListEntriesResponse_builder{}

	categorySlug, err := slugconv.FromCategoryPath(req.Msg.GetParent())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("failed to resolve category path %q: %w", req.Msg.GetParent(), err))
	}

	paginated := false
	col := s.newCollector(ctx)

	// prepopulate entry list
	col.OnHTML("body", func(body *colly.HTMLElement) {
		ct := body.DOM.Find(rowCSSSelector).Length()
		bldr.Results = slices.Grow(bldr.Results, ct)
	})

	// categories are only paginated if they have a #scroll element
	col.OnHTML("#scroll", func(_ *colly.HTMLElement) { paginated = true })

	s.scrapeRows(ctx, col, categorySlug, slugconv.ToEntryPath, func(
		kind eratov1.Entry_Kind,
		lastUpdated time.Time,
		chapterPath string,
		displayName string,
	) {
		bldr.Results = append(bldr.Results, eratov1.Entry_builder{
			Path:        chapterPath,
			DisplayName: displayName,
			Kind:        kind,
			UpdateTime:  timestamppb.New(lastUpdated),
		}.Build())
	})

	page := eratov1.ListEntriesPaginationToken_builder{Page: 1}.Build()
	if tkn := req.Msg.GetPageToken(); tkn != "" {
		if err := pagination.FromToken(tkn, page); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("malformed pagination token: %w", err))
		}
		paginated = page.GetPage() > 1
	}

	addr := s.base.JoinPath(categorySlug)
	if page.GetPage() > 1 {
		addr = addr.JoinPath(fmt.Sprintf("index%d.html", page.GetPage()-1))
	}

	if err := col.Visit(addr.String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to scrape %v: %w", addr, err))
	}

	if paginated {
		// Set Page to current page, not +1. The Paginator will increment Page
		// when the current upstream page is exhausted (all entries consumed).
		nextPage := eratov1.ListEntriesPaginationToken_builder{
			Page:            page.GetPage(),
			AfterEntry:      "categories/xxx/entries/xxx",
			StartUpdateTime: timestamppb.Now(),
		}
		if len(bldr.Results) > 0 {
			last := bldr.Results[len(bldr.Results)-1]
			nextPage.AfterEntry = last.GetPath()
			nextPage.StartUpdateTime = last.GetUpdateTime()
		}
		tkn, err := pagination.ToToken(nextPage.Build())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to construct pagination token: %w", err))
		}
		bldr.NextPageToken = tkn
	}

	return connect.NewResponse(bldr.Build()), nil
}

// GetEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) GetEntry(
	ctx context.Context,
	req *connect.Request[eratov1.GetEntryRequest],
) (*connect.Response[eratov1.Entry], error) {
	bldr := eratov1.Entry_builder{
		Path: req.Msg.GetPath(),
		Kind: eratov1.Entry_STORY,
	}

	slug, err := slugconv.FromEntryPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	bldr.DisplayName = slugconv.ToTitle(slug)

	col := s.newCollector(ctx)
	s.scrapeLastModifiedHeader(ctx, col, func(timestamp time.Time) {
		bldr.UpdateTime = timestamppb.New(timestamp)
	})
	col.OnHTML("body", func(body *colly.HTMLElement) {
		if body.DOM.Find(rowCSSSelector).Length() > 0 {
			bldr.Kind = eratov1.Entry_ANTHOLOGY
		}
	})

	if err := col.Visit(s.base.JoinPath(slug).String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(bldr.Build()), nil
}

// ListChapters satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) ListChapters(
	ctx context.Context,
	req *connect.Request[eratov1.ListChaptersRequest],
) (*connect.Response[eratov1.ListChaptersResponse], error) {
	bldr := eratov1.ListChaptersResponse_builder{}

	entrySlug, err := slugconv.FromEntryPath(req.Msg.GetParent())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	col := s.newCollector(ctx)

	// prepopulate chapter list
	col.OnHTML("body", func(body *colly.HTMLElement) {
		ct := body.DOM.Find(rowCSSSelector).Length()
		bldr.Results = slices.Grow(bldr.Results, ct)
	})

	s.scrapeRows(ctx, col, entrySlug, slugconv.ToChapterPath, func(
		_ eratov1.Entry_Kind,
		lastUpdated time.Time,
		chapterPath string,
		displayName string,
	) {
		bldr.Results = append(bldr.Results, eratov1.Chapter_builder{
			Path:        chapterPath,
			DisplayName: displayName,
			UpdateTime:  timestamppb.New(lastUpdated),
		}.Build())
	})

	slug, err := slugconv.FromEntryPath(req.Msg.GetParent())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	addr := s.base.JoinPath(slug)

	if err = col.Visit(addr.String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(bldr.Build()), nil
}

// GetChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) GetChapter(
	ctx context.Context,
	req *connect.Request[eratov1.GetChapterRequest],
) (*connect.Response[eratov1.Chapter], error) {
	bldr := eratov1.Chapter_builder{
		Path: req.Msg.GetPath(),
	}

	slug, err := slugconv.FromChapterPath(req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	bldr.DisplayName = slugconv.ToTitle(slug)

	col := s.newCollector(ctx)
	s.scrapeLastModifiedHeader(ctx, col, func(timestamp time.Time) {
		bldr.UpdateTime = timestamppb.New(timestamp)
	})

	if err := col.Visit(s.base.JoinPath(slug).String()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(bldr.Build()), nil
}

// ReadEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) ReadEntry(
	ctx context.Context,
	req *connect.Request[eratov1.ReadEntryRequest],
) (*connect.Response[eratov1.ReadEntryResponse], error) {
	res := connect.NewResponse(&eratov1.ReadEntryResponse{})
	out, err := s.readContent(ctx, slugconv.FromEntryPath, req.Msg)
	if err != nil {
		return nil, err
	}
	res.Msg.SetContent(out)
	return res, nil
}

// ReadChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (s *Scraper) ReadChapter(
	ctx context.Context,
	req *connect.Request[eratov1.ReadChapterRequest],
) (*connect.Response[eratov1.ReadChapterResponse], error) {
	res := connect.NewResponse(&eratov1.ReadChapterResponse{})
	out, err := s.readContent(ctx, slugconv.FromChapterPath, req.Msg)
	if err != nil {
		return nil, err
	}
	res.Msg.SetContent(out)
	return res, nil
}

type contentReader interface {
	GetPath() string
	GetMimeType() eratov1.ReadEntryRequest_MimeType
}

func (s *Scraper) readContent(
	ctx context.Context,
	pathToSlug func(string) (string, error),
	msg contentReader,
) (string, error) {
	slug, err := pathToSlug(msg.GetPath())
	if err != nil {
		return "", connect.NewError(connect.CodeInvalidArgument, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base.JoinPath(slug).String(), nil)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	} else if res.StatusCode != http.StatusOK {
		return "", connect.NewError(connect.CodeNotFound, nil)
	}
	defer func() { _ = res.Body.Close() }() // error is not actionable after read

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	}

	output, err := content.Transform(
		res.Header.Get("Content-Type"),
		msg.GetMimeType(),
		body,
	)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, err)
	}
	return string(output), nil
}

func (s *Scraper) newCollector(ctx context.Context) *colly.Collector {
	col := colly.NewCollector(
		colly.IgnoreRobotsTxt(),
		colly.UserAgent(userAgent),
		colly.StdlibContext(ctx),
	)
	col.SetClient(s.client)
	return col
}

func (s *Scraper) scrapeRows(
	ctx context.Context,
	col *colly.Collector,
	parentSlug string,
	childSlugToPath func(string) (string, error),
	onSuccess func(eratov1.Entry_Kind, time.Time, string, string),
) {
	parentLastModified := time.Now()
	s.scrapeLastModifiedHeader(ctx, col, func(timestamp time.Time) {
		parentLastModified = timestamp
	})

	col.OnHTML(rowCSSSelector, func(el *colly.HTMLElement) {
		kind, lastUpdated, slug, title, err := s.parseRow(el, parentLastModified, parentSlug)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to parse row",
				slog.String("element", el.Text),
				slog.Any("error", err),
			)
			return
		}

		childPath, err := childSlugToPath(slug)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to resolve path",
				slog.String("slug", slug),
				slog.Any("error", err),
			)
			return
		}

		onSuccess(kind, lastUpdated, childPath, title)
	})
}

func (s *Scraper) parseRow(row *colly.HTMLElement, parentLastUpdated time.Time, slugBase string) (
	kind eratov1.Entry_Kind,
	lastUpdated time.Time,
	slug string,
	title string,
	err error,
) {
	child := row.DOM.Children().First()
	if child.Text() == "Dir" {
		kind = eratov1.Entry_ANTHOLOGY
	} else {
		kind = eratov1.Entry_STORY
	}

	child = child.Next()
	lastUpdated, err = s.parseRowTimestamp(child.Text(), parentLastUpdated)
	if err != nil {
		return kind, lastUpdated, slug, title, fmt.Errorf("bad row timestamp %q: %w", child.Text(), err)
	}

	child = child.Next().Find("a")
	slug = child.AttrOr("href", "")
	if slug == "" {
		return kind, lastUpdated, slug, title, fmt.Errorf("bad row slug %q", child.Text())
	}
	title = slugconv.ToTitle(slug)
	slug = path.Join(slugBase, slug)

	return kind, lastUpdated, slug, title, nil
}

func (s *Scraper) scrapeLastModifiedHeader(ctx context.Context, col *colly.Collector, onSuccess func(time.Time)) {
	col.OnResponseHeaders(func(resp *colly.Response) {
		hdr := resp.Headers.Get("Last-Modified")
		lastModified, err := time.Parse(time.RFC1123, hdr)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to parse last-modified header",
				slog.String("header", hdr),
				slog.Any("error", err),
			)
		} else {
			onSuccess(lastModified)
		}
	})
}

func (s *Scraper) parseRowTimestamp(rowTimestamp string, parentLastUpdated time.Time) (time.Time, error) {
	const (
		// more recent rows leave off the year (~ within the last 12 months)
		recentRowFormat = "Jan _2 15:04"
		// older rows leave off the time (~ older than a year)
		olderRowFormat = "Jan _2 2006"
	)
	parsed, err := time.ParseInLocation(recentRowFormat, rowTimestamp, s.locale)
	if err != nil {
		// must be in the older format
		return time.ParseInLocation(olderRowFormat, rowTimestamp, s.locale)
	}

	// Since the year is not included in the recent timestamps, we need to
	// divine it from the parent's last modified date and the current timestamp.
	var year int
	switch parent, now := parentLastUpdated, time.Now(); {
	case parent.Year() < now.Year(): // we are in a future year relative to the page
		year = parent.Year()
		if parent.Month() < parsed.Month() { // crossed back another year
			year--
		}
	case now.Month() < parsed.Month(): // the same year, but crossed into the previous year
		year = now.Year() - 1
	default: // the same year, no crossover
		year = now.Year()
	}
	return time.Date(
		year,
		parsed.Month(),
		parsed.Day(),
		parsed.Hour(),
		parsed.Minute(),
		parsed.Second(),
		0,
		parsed.Location(),
	), nil
}

var _ eratov1connect.ArchiveServiceHandler = (*Scraper)(nil)
