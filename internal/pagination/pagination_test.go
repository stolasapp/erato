package pagination

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

func TestToToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     proto.Message
		wantErr bool
	}{
		{
			name: "valid message",
			msg: eratov1.ListCategoriesPaginationToken_builder{
				AfterCategory: "categories/foo",
			}.Build(),
			wantErr: false,
		},
		{
			name:    "invalid message missing required field",
			msg:     &eratov1.ListCategoriesPaginationToken{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tkn, err := ToToken(tt.msg)
			if tt.wantErr {
				var tokenErr TokenError
				require.ErrorAs(t, err, &tokenErr)
				assert.Empty(t, tkn)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, tkn)
			}
		})
	}
}

func TestFromToken(t *testing.T) {
	t.Parallel()

	// Create a valid token for testing
	validMsg := eratov1.ListCategoriesPaginationToken_builder{
		AfterCategory: "categories/foo",
	}.Build()
	validToken, err := ToToken(validMsg)
	require.NoError(t, err)

	// Create a token with valid base64/proto but missing required fields
	emptyMsg := &eratov1.ListCategoriesPaginationToken{}
	emptyProtoBytes, err := proto.Marshal(emptyMsg)
	require.NoError(t, err)
	invalidValidationToken := tokenEncoding.EncodeToString(emptyProtoBytes)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			token:   "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "valid base64 invalid proto",
			token:   tokenEncoding.EncodeToString([]byte("not a protobuf")),
			wantErr: true,
		},
		{
			name:    "valid proto fails validation",
			token:   invalidValidationToken,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &eratov1.ListCategoriesPaginationToken{}
			err := FromToken(tt.token, out)
			if tt.wantErr {
				var tokenErr TokenError
				require.ErrorAs(t, err, &tokenErr)
			} else {
				require.NoError(t, err)
				assert.True(t, proto.Equal(validMsg, out))
			}
		})
	}
}

func TestTokenRoundTrip(t *testing.T) {
	t.Parallel()

	msg := eratov1.ListCategoriesPaginationToken_builder{
		AfterCategory: "categories/bar",
	}.Build()

	tkn, err := ToToken(msg)
	require.NoError(t, err)

	out := &eratov1.ListCategoriesPaginationToken{}
	err = FromToken(tkn, out)
	require.NoError(t, err)

	assert.True(t, proto.Equal(msg, out), "expected %v, got %v", msg, out)
}

func TestTokenErrorMessage(t *testing.T) {
	t.Parallel()

	err := TokenError{cause: errors.New("underlying cause")}
	assert.Equal(t, "invalid pagination token", err.Error())
	assert.EqualError(t, errors.Unwrap(err), "underlying cause")
}
