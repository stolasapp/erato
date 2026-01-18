// Package pagination provides utilities around page tokens.
package pagination

import (
	"encoding/base64"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
)

var tokenEncoding = base64.RawURLEncoding

// TokenError is an opaque error related to pagination tokens. The error message
// does not reveal internal details; use [errors.Unwrap] to access the cause.
type TokenError struct {
	cause error
}

// Error satisfies [error].
func (terr TokenError) Error() string {
	return "invalid pagination token"
}

// Unwrap returns the underlying cause of the token error.
func (terr TokenError) Unwrap() error {
	return terr.cause
}

// FromToken decodes an opaque pagination token into the provided proto message.
// Returns a [TokenError] if decoding or validation fails.
func FromToken(tkn string, msg proto.Message) error {
	data, err := tokenEncoding.DecodeString(tkn)
	if err != nil {
		return TokenError{cause: err}
	}
	if err = proto.Unmarshal(data, msg); err != nil {
		return TokenError{cause: err}
	}
	if err = protovalidate.Validate(msg); err != nil {
		return TokenError{cause: err}
	}
	return nil
}

// ToToken encodes a proto message into an opaque pagination token. Returns a
// [TokenError] if validation or encoding fails.
func ToToken(msg proto.Message) (string, error) {
	if err := protovalidate.Validate(msg); err != nil {
		return "", TokenError{cause: err}
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return "", TokenError{cause: err}
	}
	return tokenEncoding.EncodeToString(data), nil
}
