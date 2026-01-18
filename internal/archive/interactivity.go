package archive

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/sec"
	"github.com/stolasapp/erato/internal/storage"
	"github.com/stolasapp/erato/internal/storage/db"
)

// Interactivity is an [eratov1connect.ArchiveServiceHandler] decorator that
// implements the update methods for resources.
type Interactivity struct {
	eratov1connect.ArchiveServiceHandler

	store storage.Resources
}

// NewInteractivity wraps inner and updates resource information in the provided
// store.
func NewInteractivity(inner eratov1connect.ArchiveServiceHandler, store storage.Resources) Interactivity {
	return Interactivity{
		ArchiveServiceHandler: inner,
		store:                 store,
	}
}

// UpdateCategory satisfies [eratov1connect.ArchiveServiceHandler].
func (i Interactivity) UpdateCategory(
	ctx context.Context,
	req *connect.Request[eratov1.UpdateCategoryRequest],
) (*connect.Response[eratov1.Category], error) {
	if err := updateResource(
		ctx,
		i.store,
		req,
		(*eratov1.UpdateCategoryRequest).GetCategory,
		func(resource *db.Resource, category *eratov1.Category, mask *fieldmaskpb.FieldMask) error {
			for _, path := range mask.GetPaths() {
				switch strings.ToLower(path) {
				case "hidden":
					resource.Hidden = category.GetHidden()
				default:
					return connect.NewError(connect.CodeInvalidArgument, nil)
				}
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return i.GetCategory(ctx, connect.NewRequest(eratov1.GetCategoryRequest_builder{
		Path: req.Msg.GetPath(),
	}.Build()))
}

// UpdateEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (i Interactivity) UpdateEntry(
	ctx context.Context,
	req *connect.Request[eratov1.UpdateEntryRequest],
) (*connect.Response[eratov1.Entry], error) {
	if err := updateResource(
		ctx,
		i.store,
		req,
		(*eratov1.UpdateEntryRequest).GetEntry,
		func(resource *db.Resource, entry *eratov1.Entry, mask *fieldmaskpb.FieldMask) error {
			for _, path := range mask.GetPaths() {
				switch strings.ToLower(path) {
				case "hidden":
					resource.Hidden = entry.GetHidden()
				case "starred":
					resource.Starred = entry.GetStarred()
				case "view_time":
					resource.ViewTime = sql.NullTime{
						Valid: entry.HasViewTime(),
						Time:  entry.GetViewTime().AsTime(),
					}
				case "read_time":
					resource.ReadTime = sql.NullTime{
						Valid: entry.HasReadTime(),
						Time:  entry.GetReadTime().AsTime(),
					}
				default:
					return connect.NewError(connect.CodeInvalidArgument, nil)
				}
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return i.GetEntry(ctx, connect.NewRequest(eratov1.GetEntryRequest_builder{
		Path: req.Msg.GetPath(),
	}.Build()))
}

// UpdateChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (i Interactivity) UpdateChapter(
	ctx context.Context,
	req *connect.Request[eratov1.UpdateChapterRequest],
) (*connect.Response[eratov1.Chapter], error) {
	if err := updateResource(
		ctx,
		i.store,
		req,
		(*eratov1.UpdateChapterRequest).GetChapter,
		func(resource *db.Resource, chapter *eratov1.Chapter, mask *fieldmaskpb.FieldMask) error {
			for _, path := range mask.GetPaths() {
				switch strings.ToLower(path) {
				case "view_time":
					resource.ViewTime = sql.NullTime{
						Valid: chapter.HasViewTime(),
						Time:  chapter.GetViewTime().AsTime(),
					}
				case "read_time":
					resource.ReadTime = sql.NullTime{
						Valid: chapter.HasReadTime(),
						Time:  chapter.GetReadTime().AsTime(),
					}
				default:
					return connect.NewError(connect.CodeInvalidArgument, nil)
				}
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return i.GetChapter(ctx, connect.NewRequest(eratov1.GetChapterRequest_builder{
		Path: req.Msg.GetPath(),
	}.Build()))
}

func updateResource[
	Req any,
	Res proto.Message,
	ReqP interface {
		*Req
		GetPath() string
		GetUpdateMask() *fieldmaskpb.FieldMask
	},
](
	ctx context.Context,
	store storage.Resources,
	req *connect.Request[Req],
	get func(ReqP) Res,
	update func(*db.Resource, Res, *fieldmaskpb.FieldMask) error,
) error {
	user := sec.GetAuthenticatedUser(ctx)
	var msg ReqP = req.Msg
	dbRes, err := store.GetResource(ctx, user.ID, msg.GetPath())
	if errors.Is(err, storage.ErrNotFound) {
		// user hasn't interacted with this resource yet
		dbRes = db.Resource{
			User: user.ID,
			Path: msg.GetPath(),
		}
	} else if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	resource := get(msg)
	mask := msg.GetUpdateMask()
	if !mask.IsValid(resource) {
		return connect.NewError(connect.CodeInvalidArgument, nil)
	} else if len(mask.GetPaths()) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, nil)
	}

	if err = update(&dbRes, resource, mask); err != nil {
		return err
	} else if err = store.UpsertResource(ctx, dbRes); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	return nil
}

var _ eratov1connect.ArchiveServiceHandler = Interactivity{}
