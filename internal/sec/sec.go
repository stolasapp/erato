// Package sec provides authentication and security primitives for the ConnectRPC
// service and web application.
//
// # Authentication
//
// Authentication uses HTTP Basic Auth via the connectrpc.com/authn middleware.
// Credentials are validated against bcrypt password hashes stored in the database.
//
// IMPORTANT: Basic Auth transmits credentials in base64 encoding (not encrypted).
// TLS must be used in production to protect credentials in transit.
//
// # Components
//
//   - [Authenticate]: Validates Basic Auth credentials against the user store
//   - [NewConnectAuthMiddleware]: Creates ConnectRPC middleware for authentication
//   - [GetAuthenticatedUser], [SetAuthenticatedUser]: Context accessors for user info
//   - [HashPassword], [ComparePassword]: bcrypt password hashing utilities
package sec
