package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrubTextDocument(t *testing.T) {
	t.Parallel()
	scrub := ScrubTextDocument()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "plain text unchanged",
			input: "Hello, world!",
			want:  "Hello, world!",
		},

		// Line ending normalization
		{
			name:  "CRLF normalized to LF",
			input: "line1\r\nline2\r\nline3",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "CR normalized to LF",
			input: "line1\rline2\rline3",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "mixed line endings normalized",
			input: "line1\r\nline2\rline3\nline4",
			want:  "line1\nline2\nline3\nline4",
		},

		// Paragraph whitespace
		{
			name:  "two space indent gets blank line",
			input: "First paragraph.\n  Second paragraph.",
			want:  "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:  "tab indent gets blank line",
			input: "First paragraph.\n\tSecond paragraph.",
			want:  "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:  "single space does not get blank line",
			input: "First paragraph.\n Second paragraph.",
			want:  "First paragraph.\nSecond paragraph.",
		},
		{
			name:  "leading whitespace removed",
			input: "  indented line",
			want:  "indented line",
		},
		{
			name:  "leading tab removed",
			input: "\tindented line",
			want:  "indented line",
		},

		// Trailing whitespace
		{
			name:  "trailing spaces removed",
			input: "line with trailing spaces   \nclean line",
			want:  "line with trailing spaces\nclean line",
		},
		{
			name:  "trailing tabs removed",
			input: "line with trailing tab\t\nclean line",
			want:  "line with trailing tab\nclean line",
		},

		// Decorative HR - asterisks
		{
			name:  "three asterisks normalized",
			input: "text\n***\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five asterisks normalized",
			input: "text\n*****\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "spaced asterisks normalized",
			input: "text\n* * *\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "two asterisks not normalized",
			input: "text\n**\nmore text",
			want:  "text\n**\nmore text",
		},
		{
			name:  "asterisks with preceding text not normalized",
			input: "text ***\nmore text",
			want:  "text ***\nmore text",
		},

		// Decorative HR - dashes
		{
			name:  "three dashes normalized",
			input: "text\n---\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five dashes normalized",
			input: "text\n-----\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "two dashes not normalized",
			input: "text\n--\nmore text",
			want:  "text\n--\nmore text",
		},

		// Decorative HR - underscores
		{
			name:  "three underscores normalized",
			input: "text\n___\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five underscores normalized",
			input: "text\n_____\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "two underscores not normalized",
			input: "text\n__\nmore text",
			want:  "text\n__\nmore text",
		},

		// Decorative HR - tildes (code fence chars)
		{
			name:  "three tildes normalized",
			input: "text\n~~~\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five tildes normalized",
			input: "text\n~~~~~\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "two tildes not normalized",
			input: "text\n~~\nmore text",
			want:  "text\n~~\nmore text",
		},

		// Decorative HR - backticks (code fence chars)
		{
			name:  "three backticks normalized",
			input: "text\n```\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five backticks normalized",
			input: "text\n`````\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "two backticks not normalized",
			input: "text\n``\nmore text",
			want:  "text\n``\nmore text",
		},

		// Decorative HR - equals
		{
			name:  "three equals normalized",
			input: "text\n===\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five equals normalized",
			input: "text\n=====\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "two equals not normalized",
			input: "text\n==\nmore text",
			want:  "text\n==\nmore text",
		},

		// Decorative HR - hashes
		{
			name:  "three hashes alone normalized",
			input: "text\n###\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "five hashes alone normalized",
			input: "text\n#####\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "hash heading not normalized",
			input: "### Heading\n\nParagraph",
			want:  "### Heading\n\nParagraph",
		},
		{
			name:  "two hashes not normalized",
			input: "text\n##\nmore text",
			want:  "text\n##\nmore text",
		},
		{
			name:  "hashes with preceding text not normalized",
			input: "text ###\nmore text",
			want:  "text ###\nmore text",
		},

		// HR with leading/trailing whitespace on line (should still match)
		{
			name:  "HR with leading whitespace normalized",
			input: "text\n   ***\nmore text",
			want:  "text\n---\nmore text",
		},
		{
			name:  "HR with trailing whitespace normalized",
			input: "text\n***   \nmore text",
			want:  "text\n---\nmore text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := scrub([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}
