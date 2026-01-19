package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:maintidx // large table-driven test
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

		// Isolated dash prefix removal (dialog markers)
		{
			name:  "isolated dash prefix stripped",
			input: "some text\n- dialog line\nmore text",
			want:  "some text\ndialog line\nmore text",
		},
		{
			name:  "isolated dash with blank lines stripped",
			input: "some text\n\n- dialog line\n\nmore text",
			want:  "some text\n\ndialog line\n\nmore text",
		},
		{
			name:  "consecutive dash lines preserved as list",
			input: "some text\n- item 1\n- item 2\nmore text",
			want:  "some text\n- item 1\n- item 2\nmore text",
		},
		{
			name:  "dash lines with blank between preserved as list",
			input: "some text\n- item 1\n\n- item 2\nmore text",
			want:  "some text\n- item 1\n\n- item 2\nmore text",
		},
		{
			name:  "multiple isolated dashes stripped",
			input: "text\n- dialog 1\nnarration\n- dialog 2\nmore text",
			want:  "text\ndialog 1\nnarration\ndialog 2\nmore text",
		},
		{
			name:  "dash at start of document isolated",
			input: "- opening line\nsome text",
			want:  "opening line\nsome text",
		},
		{
			name:  "dash at end of document isolated",
			input: "some text\n- closing line",
			want:  "some text\nclosing line",
		},
		{
			name:  "three consecutive dashes preserved",
			input: "text\n- one\n- two\n- three\nmore",
			want:  "text\n- one\n- two\n- three\nmore",
		},
		{
			name:  "isolated dash without space stripped",
			input: "some text\n-\"dialog line\"\nmore text",
			want:  "some text\n\"dialog line\"\nmore text",
		},
		{
			name:  "consecutive dashes without space preserved",
			input: "text\n-\"item 1\"\n-\"item 2\"\nmore",
			want:  "text\n-\"item 1\"\n-\"item 2\"\nmore",
		},
		{
			name:  "mixed dash styles treated as list",
			input: "text\n- item 1\n-\"item 2\"\nmore",
			want:  "text\n- item 1\n-\"item 2\"\nmore",
		},
		{
			name:  "double dash not treated as dialog",
			input: "text\n--separator\nmore",
			want:  "text\n--separator\nmore",
		},

		// Email header wrapping
		{
			name:  "email headers wrapped in details",
			input: "From: Author <a@b.com>\nSubject: Test\n\nBody text here.",
			want: "<details class=\"email-headers\">\n<summary>Email headers</summary>\n" +
				"From: Author &lt;a@b.com&gt;<br>\nSubject: Test\n</details>\n\nBody text here.",
		},
		{
			name:  "email headers with HR separator",
			input: "Date: Wed, 14 Jan 2026\nFrom: Author\n---\n\nBody text.",
			want: "<details class=\"email-headers\">\n<summary>Email headers</summary>\n" +
				"Date: Wed, 14 Jan 2026<br>\nFrom: Author\n</details>\n\nBody text.",
		},
		{
			name:  "no email headers unchanged",
			input: "Just regular text\nwith multiple lines.",
			want:  "Just regular text\nwith multiple lines.",
		},
		{
			name:  "single header line",
			input: "Subject: Hello\n\nBody.",
			want: "<details class=\"email-headers\">\n<summary>Email headers</summary>\n" +
				"Subject: Hello\n</details>\n\nBody.",
		},
		{
			name:  "headers must be at start",
			input: "Some intro\nFrom: Author\n\nBody.",
			want:  "Some intro\nFrom: Author\n\nBody.",
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
