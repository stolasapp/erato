package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
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
		{
			name:        "UTF-8 with high bytes unchanged",
			contentType: "text/plain",
			input:       []byte("Héllo wörld with ümlauts"),
			want:        "Héllo wörld with ümlauts",
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

func TestUTF8TransformerWithExplicitCharset(t *testing.T) {
	t.Parallel()

	t.Run("windows-1252 with explicit charset", func(t *testing.T) {
		t.Parallel()
		// Windows-1252 encoded "curly quotes" (0x93, 0x94)
		input := []byte{0x93, 'H', 'e', 'l', 'l', 'o', 0x94}
		transformer := UTF8Transformer("text/plain; charset=windows-1252")
		got, err := transformer(input)
		require.NoError(t, err)
		// 0x93 and 0x94 are left/right double quotes in Windows-1252
		assert.Equal(t, "\u201cHello\u201d", string(got))
	})

	t.Run("iso-8859-1 with explicit charset", func(t *testing.T) {
		t.Parallel()
		// ISO-8859-1 encoded "café" (0xe9 = é)
		input := []byte{'c', 'a', 'f', 0xe9}
		transformer := UTF8Transformer("text/plain; charset=iso-8859-1")
		got, err := transformer(input)
		require.NoError(t, err)
		assert.Equal(t, "café", string(got))
	})
}

func TestUTF8TransformerStatisticalDetection(t *testing.T) {
	t.Parallel()

	t.Run("detects windows-1252 in plain text without charset", func(t *testing.T) {
		t.Parallel()
		// Create Windows-1252 encoded text with characteristic bytes
		// that chardet can statistically detect.
		// Using curly quotes and other Windows-1252 specific chars.
		win1252Text := encodeWindows1252(t,
			`"This is a story," she said. "It has 'curly quotes' and em-dashes—like this."

The café was quiet. "Would you like some more?" asked the maître d'.

"Yes, please," I replied. The résumé lay on the table.

She smiled. "That'll be €5.00."`)

		transformer := UTF8Transformer("text/plain") // No charset hint
		got, err := transformer(win1252Text)
		require.NoError(t, err)

		// Should be converted to proper UTF-8
		assert.Contains(t, string(got), "café")
		assert.Contains(t, string(got), "résumé")
	})
}

// encodeWindows1252 encodes a UTF-8 string to Windows-1252 bytes for testing.
func encodeWindows1252(t *testing.T, s string) []byte {
	t.Helper()
	encoder := charmap.Windows1252.NewEncoder()
	out, err := encoder.Bytes([]byte(s))
	require.NoError(t, err, "failed to encode test string to Windows-1252")
	return out
}
