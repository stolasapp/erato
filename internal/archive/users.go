package archive

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/sec"
	"github.com/stolasapp/erato/internal/storage"
	"github.com/stolasapp/erato/internal/storage/db"
)

// Users is an [eratov1connect.ArchiveServiceHandler] decorator to handle user
// CRUD operations. This decorator should be attached inside the [Paginator] to
// ensure ListUsers paginates correctly.
type Users struct {
	eratov1connect.ArchiveServiceHandler

	store storage.Users
}

// NewUsers wraps inner and uses the provided store to handle user operations.
func NewUsers(inner eratov1connect.ArchiveServiceHandler, store storage.Users) Users {
	return Users{
		ArchiveServiceHandler: inner,
		store:                 store,
	}
}

// CreateUser satisfies [eratov1connect.ArchiveServiceHandler].
func (u Users) CreateUser(
	ctx context.Context,
	req *connect.Request[eratov1.CreateUserRequest],
) (*connect.Response[eratov1.User], error) {
	hash, err := sec.HashPassword(req.Msg.GetUser().GetPassword())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user := db.User{
		Name:         req.Msg.GetId(),
		PasswordHash: hash,
	}
	if err = u.store.UpsertUser(ctx, user); errors.Is(err, storage.ErrAlreadyExists) {
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(eratov1.User_builder{
		Path: user.Path(),
		Id:   user.Name,
	}.Build()), nil
}

// ListUsers satisfies [eratov1connect.ArchiveServiceHandler].
func (u Users) ListUsers(
	_ context.Context,
	_ *connect.Request[eratov1.ListUsersRequest],
) (*connect.Response[eratov1.ListUsersResponse], error) {
	// TODO: support getting other users if the caller's permissions allow it
	return nil, connect.NewError(connect.CodePermissionDenied, nil)
}

// GetUser satisfies [eratov1connect.ArchiveServiceHandler].
func (u Users) GetUser(
	ctx context.Context,
	req *connect.Request[eratov1.GetUserRequest],
) (*connect.Response[eratov1.User], error) {
	// TODO: support getting other users if the caller's permissions allow it
	user := sec.GetAuthenticatedUser(ctx)
	if req.Msg.GetPath() != user.Path() {
		return nil, connect.NewError(connect.CodePermissionDenied, nil)
	}

	return connect.NewResponse(eratov1.User_builder{
		Path: user.Path(),
		Id:   user.Name,
	}.Build()), nil
}

// UpdateUser satisfies [eratov1connect.ArchiveServiceHandler].
func (u Users) UpdateUser(
	ctx context.Context,
	req *connect.Request[eratov1.UpdateUserRequest],
) (*connect.Response[eratov1.User], error) {
	// only allowed to update yourself
	authd := sec.GetAuthenticatedUser(ctx)
	if req.Msg.GetPath() != authd.Path() {
		return nil, connect.NewError(connect.CodePermissionDenied, nil)
	}

	user := eratov1.User_builder{
		Path: authd.Path(),
		Id:   authd.Name,
	}.Build()

	mask := req.Msg.GetUpdateMask()
	if !mask.IsValid(user) {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	} else if len(mask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	for _, path := range mask.GetPaths() {
		switch strings.ToLower(path) {
		case "password":
			hash, err := sec.HashPassword(req.Msg.GetUser().GetPassword())
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			authd.PasswordHash = hash
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, nil)
		}
	}
	if err := u.store.UpsertUser(ctx, authd); errors.Is(err, storage.ErrAlreadyExists) {
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(user), nil
}

// DeleteUser satisfies [eratov1connect.ArchiveServiceHandler].
func (u Users) DeleteUser(
	ctx context.Context,
	req *connect.Request[eratov1.DeleteUserRequest],
) (*connect.Response[emptypb.Empty], error) {
	// only allowed to delete yourself
	authd := sec.GetAuthenticatedUser(ctx)
	if req.Msg.GetPath() != authd.Path() {
		return nil, connect.NewError(connect.CodePermissionDenied, nil)
	}

	if err := u.store.DeleteUser(ctx, authd.ID); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

var _ eratov1connect.ArchiveServiceHandler = Users{}
