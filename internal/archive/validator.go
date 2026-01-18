package archive

import (
	"context"
	"log/slog"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
)

// Validator is an [eratov1connect.ArchiveServiceHandler] decorator that validates
// requests and responses using protovalidate.
type Validator struct {
	eratov1connect.ArchiveServiceHandler

	logger    *slog.Logger
	validator protovalidate.Validator
}

// NewValidator wraps inner and validates all requests and responses.
// Invalid requests return InvalidArgument errors. Invalid responses are logged
// as warnings but still returned.
func NewValidator(inner eratov1connect.ArchiveServiceHandler, logger *slog.Logger) (*Validator, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return &Validator{
		ArchiveServiceHandler: inner,
		logger:                logger,
		validator:             validator,
	}, nil
}

// ListCategories satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) ListCategories(
	ctx context.Context, req *connect.Request[eratov1.ListCategoriesRequest],
) (*connect.Response[eratov1.ListCategoriesResponse], error) {
	return validate(ctx, v, "ListCategories", req, v.ArchiveServiceHandler.ListCategories)
}

// GetCategory satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) GetCategory(
	ctx context.Context, req *connect.Request[eratov1.GetCategoryRequest],
) (*connect.Response[eratov1.Category], error) {
	return validate(ctx, v, "GetCategory", req, v.ArchiveServiceHandler.GetCategory)
}

// UpdateCategory satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) UpdateCategory(
	ctx context.Context, req *connect.Request[eratov1.UpdateCategoryRequest],
) (*connect.Response[eratov1.Category], error) {
	return validate(ctx, v, "UpdateCategory", req, v.ArchiveServiceHandler.UpdateCategory)
}

// ListEntries satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) ListEntries(
	ctx context.Context, req *connect.Request[eratov1.ListEntriesRequest],
) (*connect.Response[eratov1.ListEntriesResponse], error) {
	return validate(ctx, v, "ListEntries", req, v.ArchiveServiceHandler.ListEntries)
}

// GetEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) GetEntry(
	ctx context.Context, req *connect.Request[eratov1.GetEntryRequest],
) (*connect.Response[eratov1.Entry], error) {
	return validate(ctx, v, "GetEntry", req, v.ArchiveServiceHandler.GetEntry)
}

// UpdateEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) UpdateEntry(
	ctx context.Context, req *connect.Request[eratov1.UpdateEntryRequest],
) (*connect.Response[eratov1.Entry], error) {
	return validate(ctx, v, "UpdateEntry", req, v.ArchiveServiceHandler.UpdateEntry)
}

// ListChapters satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) ListChapters(
	ctx context.Context, req *connect.Request[eratov1.ListChaptersRequest],
) (*connect.Response[eratov1.ListChaptersResponse], error) {
	return validate(ctx, v, "ListChapters", req, v.ArchiveServiceHandler.ListChapters)
}

// GetChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) GetChapter(
	ctx context.Context, req *connect.Request[eratov1.GetChapterRequest],
) (*connect.Response[eratov1.Chapter], error) {
	return validate(ctx, v, "GetChapter", req, v.ArchiveServiceHandler.GetChapter)
}

// UpdateChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) UpdateChapter(
	ctx context.Context, req *connect.Request[eratov1.UpdateChapterRequest],
) (*connect.Response[eratov1.Chapter], error) {
	return validate(ctx, v, "UpdateChapter", req, v.ArchiveServiceHandler.UpdateChapter)
}

// ReadEntry satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) ReadEntry(
	ctx context.Context, req *connect.Request[eratov1.ReadEntryRequest],
) (*connect.Response[eratov1.ReadEntryResponse], error) {
	return validate(ctx, v, "ReadEntry", req, v.ArchiveServiceHandler.ReadEntry)
}

// ReadChapter satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) ReadChapter(
	ctx context.Context, req *connect.Request[eratov1.ReadChapterRequest],
) (*connect.Response[eratov1.ReadChapterResponse], error) {
	return validate(ctx, v, "ReadChapter", req, v.ArchiveServiceHandler.ReadChapter)
}

// CreateUser satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) CreateUser(
	ctx context.Context, req *connect.Request[eratov1.CreateUserRequest],
) (*connect.Response[eratov1.User], error) {
	return validate(ctx, v, "CreateUser", req, v.ArchiveServiceHandler.CreateUser)
}

// ListUsers satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) ListUsers(
	ctx context.Context, req *connect.Request[eratov1.ListUsersRequest],
) (*connect.Response[eratov1.ListUsersResponse], error) {
	return validate(ctx, v, "ListUsers", req, v.ArchiveServiceHandler.ListUsers)
}

// GetUser satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) GetUser(
	ctx context.Context, req *connect.Request[eratov1.GetUserRequest],
) (*connect.Response[eratov1.User], error) {
	return validate(ctx, v, "GetUser", req, v.ArchiveServiceHandler.GetUser)
}

// UpdateUser satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) UpdateUser(
	ctx context.Context, req *connect.Request[eratov1.UpdateUserRequest],
) (*connect.Response[eratov1.User], error) {
	return validate(ctx, v, "UpdateUser", req, v.ArchiveServiceHandler.UpdateUser)
}

// DeleteUser satisfies [eratov1connect.ArchiveServiceHandler].
func (v *Validator) DeleteUser(
	ctx context.Context, req *connect.Request[eratov1.DeleteUserRequest],
) (*connect.Response[emptypb.Empty], error) {
	return validate(ctx, v, "DeleteUser", req, v.ArchiveServiceHandler.DeleteUser)
}

func validate[
	Req, Res any,
	ReqP interface {
		*Req
		proto.Message
	},
	ResP interface {
		*Res
		proto.Message
	},
](
	ctx context.Context,
	validator *Validator,
	name string,
	req *connect.Request[Req],
	handle func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error),
) (*connect.Response[Res], error) {
	var reqP ReqP = req.Msg
	if err := validator.validator.Validate(reqP); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	res, err := handle(ctx, req)
	if err != nil {
		return nil, err
	}

	var resP ResP = res.Msg
	if err := validator.validator.Validate(resP); err != nil {
		validator.logger.WarnContext(ctx, "response validation failed",
			slog.Any("error", err),
			slog.String("handler", name),
		)
	}

	return res, nil
}

var _ eratov1connect.ArchiveServiceHandler = (*Validator)(nil)
