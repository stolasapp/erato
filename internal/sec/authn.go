package sec

import (
	"context"
	"net/http"

	"connectrpc.com/authn"
	"connectrpc.com/connect"

	"github.com/stolasapp/erato/internal/storage"
	"github.com/stolasapp/erato/internal/storage/db"
)

// Authenticate resolves the logged in user from req. If the information is
// invalid, a ConnectRPC error is returned.
func Authenticate(ctx context.Context, req *http.Request, store storage.Users) (user db.User, err error) {
	username, password, ok := req.BasicAuth()
	if !ok {
		return user, authn.Errorf("invalid authorization header")
	}
	if user, err = store.GetUserByName(ctx, username); err != nil {
		return user, authn.Errorf("invalid username or password")
	}
	if err = ComparePassword(password, user.PasswordHash); err != nil {
		return user, authn.Errorf("invalid username or password")
	}
	return user, nil
}

// NewConnectAuthMiddleware returns a new authentication middleware for ConnectRPC.
func NewConnectAuthMiddleware(store storage.Users, opts ...connect.HandlerOption) *authn.Middleware {
	return authn.NewMiddleware(func(ctx context.Context, req *http.Request) (any, error) {
		return Authenticate(ctx, req, store)
	}, opts...)
}

// GetAuthenticatedUser returns the user information for the authenticated user.
// Returns a zero-value User if the context has no authenticated user or if
// the stored value is not a User (should only happen if middleware is misconfigured).
func GetAuthenticatedUser(ctx context.Context) db.User {
	if user, ok := authn.GetInfo(ctx).(db.User); ok {
		return user
	}
	return db.User{}
}

// SetAuthenticatedUser sets the user information for an authenticated user. The
// authn.Middleware automatically injects this information; this function is
// provided as a convenience for testing.
func SetAuthenticatedUser(ctx context.Context, user db.User) context.Context {
	return authn.SetInfo(ctx, user)
}
