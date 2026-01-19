package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUTF8Transformer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		input       []byte
		want        string
	}{
		{
			name:        "plain text unchanged",
			contentType: "text/plain",
			input:       []byte("Hello, world!"),
			want:        "Hello, world!",
		},
		{
			name:        "UTF-8 BOM stripped",
			contentType: "text/plain; charset=utf-8",
			input:       []byte{0xEF, 0xBB, 0xBF, 'H', 'e', 'l', 'l', 'o'},
			want:        "Hello",
		},
		{
			name:        "BOM stripped with multiline content",
			contentType: "text/plain",
			input:       append([]byte{0xEF, 0xBB, 0xBF}, []byte("Line 1\nLine 2\nLine 3")...),
			want:        "Line 1\nLine 2\nLine 3",
		},
		{
			name:        "no BOM leaves content unchanged",
			contentType: "text/plain",
			input:       []byte("No BOM here"),
			want:        "No BOM here",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			transformer := UTF8Transformer(test.contentType)
			got, err := transformer(test.input)
			require.NoError(t, err)
			assert.Equal(t, test.want, string(got))
		})
	}
}
