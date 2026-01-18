package archive

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/sec"
	"github.com/stolasapp/erato/internal/storage"
	"github.com/stolasapp/erato/internal/storage/db"
)

// Hydrator is an [eratov1connect.ArchiveServiceHandler] decorator that updates
// resource information with user-specific data from storage.
type Hydrator struct {
	eratov1connect.ArchiveServiceHandler

	store storage.Resources
}

// NewHydrator wraps inner and pulls user-specific resource data from store.
func NewHydrator(inner eratov1connect.ArchiveServiceHandler, store storage.Resources) *Hydrator {
	return &Hydrator{
		ArchiveServiceHandler: inner,
		store:                 store,
	}
}

// ListCategories satisfies [eratov1connect.ArchiveServiceHandler].
func (h Hydrator) ListCategories(
	ctx context.Context,
	req *connect.Request[eratov1.ListCategoriesRequest],
) (*connect.Response[eratov1.ListCategoriesResponse], error) {
	return hydrateList(
		ctx,
		req,
		h.store,
		h.ArchiveServiceHandler.ListCategories,
		h.hydrateCategory,
	)
}

// GetCategory satisfies [eratov1connect.ArchiveServiceHandler].
func (h Hydrator) GetCategory(
	ctx context.Context,
	req *connect.Request[eratov1.GetCategoryRequest],
) (res *connect.Response[eratov1.Category], err error) {
	return hydrateResource(
		ctx,
		req,
		h.store,
		h.ArchiveServiceHandler.GetCategory,
		h.hydrateCategory,
	)
}

// ListEntries satisfies [eratov1connect.ArchiveServiceHandler].
func (h Hydrator) ListEntries(
	ctx context.Context,
	req *connect.Request[eratov1.ListEntriesRequest],
) (*connect.Response[eratov1.ListEntriesResponse], error) {
	return hydrateList(
		ctx,
		req,
		h.store,
		h.ArchiveServiceHandler.ListEntries,
		h.hydrateEntry,
	)
}

// GetEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (h Hydrator) GetEntry(
	ctx context.Context,
	req *connect.Request[eratov1.GetEntryRequest],
) (res *connect.Response[eratov1.Entry], err error) {
	return hydrateResource(
		ctx,
		req,
		h.store,
		h.ArchiveServiceHandler.GetEntry,
		h.hydrateEntry,
	)
}

// ListChapters satisfies [eratov1connect.ArchiveServiceHandler].
func (h Hydrator) ListChapters(
	ctx context.Context,
	req *connect.Request[eratov1.ListChaptersRequest],
) (*connect.Response[eratov1.ListChaptersResponse], error) {
	return hydrateList(
		ctx,
		req,
		h.store,
		h.ArchiveServiceHandler.ListChapters,
		h.hydrateChapter,
	)
}

// GetChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (h Hydrator) GetChapter(
	ctx context.Context,
	req *connect.Request[eratov1.GetChapterRequest],
) (res *connect.Response[eratov1.Chapter], err error) {
	return hydrateResource(
		ctx,
		req,
		h.store,
		h.ArchiveServiceHandler.GetChapter,
		h.hydrateChapter,
	)
}

func (h Hydrator) hydrateCategory(category *eratov1.Category, resource *db.Resource) {
	category.SetHidden(resource.Hidden)
}

func (h Hydrator) hydrateEntry(entry *eratov1.Entry, resource *db.Resource) {
	entry.SetStarred(resource.Starred)
	entry.SetHidden(resource.Hidden)
	if viewTime := resource.ViewTime; viewTime.Valid {
		entry.SetViewTime(timestamppb.New(viewTime.Time))
	}
	if readTime := resource.ReadTime; readTime.Valid {
		entry.SetReadTime(timestamppb.New(readTime.Time))
	}
}

func (h Hydrator) hydrateChapter(chapter *eratov1.Chapter, resource *db.Resource) {
	if viewTime := resource.ViewTime; viewTime.Valid {
		chapter.SetViewTime(timestamppb.New(viewTime.Time))
	}
	if readTime := resource.ReadTime; readTime.Valid {
		chapter.SetReadTime(timestamppb.New(readTime.Time))
	}
}

func hydrateList[
	Req any,
	Res any,
	ResP interface {
		*Res
		GetResults() []Item
	},
	Item interface {
		GetPath() string
	},
](
	ctx context.Context,
	req *connect.Request[Req],
	store storage.Resources,
	handle func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error),
	hydrate func(Item, *db.Resource),
) (res *connect.Response[Res], err error) {
	res, err = handle(ctx, req)
	if err != nil {
		return nil, err
	}

	var msg ResP = res.Msg
	results := msg.GetResults()
	paths := make([]string, len(results))
	lookup := make(map[string]Item, len(results))
	for i, result := range results {
		path := result.GetPath()
		paths[i] = path
		lookup[path] = result
	}

	userData, err := store.ListResources(ctx, sec.GetAuthenticatedUser(ctx).ID, paths...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for i := range userData {
		data := &userData[i]
		result, ok := lookup[data.Path]
		if !ok {
			continue
		}
		hydrate(result, data)
	}
	return res, nil
}

func hydrateResource[
	Req any,
	Res any,
	ReqP interface {
		*Req
		GetPath() string
	},
](
	ctx context.Context,
	req *connect.Request[Req],
	store storage.Resources,
	handle func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error),
	hydrate func(*Res, *db.Resource),
) (res *connect.Response[Res], err error) {
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		res, err = handle(ctx, req)
		return err
	})
	var userData db.Resource
	grp.Go(func() error {
		var msg ReqP = req.Msg
		data, err := store.GetResource(ctx, sec.GetAuthenticatedUser(ctx).ID, msg.GetPath())
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		} else if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		userData = data
		return nil
	})
	if err = grp.Wait(); err != nil {
		return nil, err
	}
	hydrate(res.Msg, &userData)
	return res, nil
}

var _ eratov1connect.ArchiveServiceHandler = (*Hydrator)(nil)
