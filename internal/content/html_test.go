package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractHTMLBody(t *testing.T) {
	t.Parallel()
	extract := ExtractHTMLBody()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extracts body content",
			input: "<html><head><title>Test</title></head><body><p>Hello</p></body></html>",
			want:  "<p>Hello</p>",
		},
		{
			name:  "returns input unchanged when no body tag",
			input: "<p>Just a paragraph</p>",
			want:  "<p>Just a paragraph</p>",
		},
		{
			name:  "extracts body discarding attributes",
			input: `<body class="main"><div>Content</div></body>`,
			want:  "<div>Content</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extract([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestNormalizeNBSP(t *testing.T) {
	t.Parallel()
	normalize := NormalizeNBSP()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "replaces nbsp entity",
			input: "hello&nbsp;world",
			want:  "hello world",
		},
		{
			name:  "replaces nbsp entity case insensitive",
			input: "hello&NBSP;world&Nbsp;test",
			want:  "hello world test",
		},
		{
			name:  "replaces unicode nbsp character",
			input: "hello\u00A0world",
			want:  "hello world",
		},
		{
			name:  "replaces multiple nbsp",
			input: "a&nbsp;&nbsp;&nbsp;b",
			want:  "a   b",
		},
		{
			name:  "handles mixed entities and unicode",
			input: "a&nbsp;\u00A0b",
			want:  "a  b",
		},
		{
			name:  "no nbsp unchanged",
			input: "hello world",
			want:  "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalize([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestScrubHTML(t *testing.T) {
	t.Parallel()
	scrub := ScrubHTML()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty inline element removal
		{
			name:  "removes empty span",
			input: "<p>Hello<span></span>World</p>",
			want:  "<p>HelloWorld</p>",
		},
		{
			name:  "removes empty em",
			input: "<p>Hello<em></em>World</p>",
			want:  "<p>HelloWorld</p>",
		},
		{
			name:  "removes whitespace-only span",
			input: "<p>Hello<span>   </span>World</p>",
			want:  "<p>HelloWorld</p>",
		},
		{
			name:  "keeps non-empty inline",
			input: "<p>Hello<strong>!</strong>World</p>",
			want:  "<p>Hello<strong>!</strong>World</p>",
		},
		{
			name:  "removes nested empty inlines",
			input: "<p>Hello<em><strong></strong></em>World</p>",
			want:  "<p>HelloWorld</p>",
		},

		// Spacer block removal (blocks containing only br and whitespace)
		{
			name:  "removes p containing only br",
			input: "<p>Hello</p><p><br></p><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "removes p containing br with whitespace",
			input: "<p>Hello</p><p> <br> </p><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "removes p containing br with newlines",
			input: "<p>Hello</p><p>\n<br>\n</p><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "removes p containing multiple brs",
			input: "<p>Hello</p><p><br><br></p><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "removes div containing only br",
			input: "<p>Hello</p><div><br></div><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "keeps p with text and br",
			input: "<p>Hello<br>World</p>",
			want:  "<p>Hello<br/>World</p>",
		},

		// Empty block replacement with BR
		{
			name:  "replaces empty div with br",
			input: "<p>Hello</p><div></div><p>World</p>",
			want:  "<p>Hello</p><br/><p>World</p>",
		},
		{
			name:  "replaces empty p with br",
			input: "<div><p></p></div>",
			want:  "<div><br/></div>",
		},
		{
			name:  "keeps non-empty block",
			input: "<div>Content</div>",
			want:  "<div>Content</div>",
		},

		// Excessive BR collapsing
		{
			name:  "two brs unchanged",
			input: "<p>Hello</p><br><br><p>World</p>",
			want:  "<p>Hello</p><br/><br/><p>World</p>",
		},
		{
			name:  "three brs collapsed to two",
			input: "<p>Hello</p><br><br><br><p>World</p>",
			want:  "<p>Hello</p><br/><br/><p>World</p>",
		},
		{
			name:  "five brs collapsed to two",
			input: "<p>Hello</p><br><br><br><br><br><p>World</p>",
			want:  "<p>Hello</p><br/><br/><p>World</p>",
		},
		{
			name:  "brs with whitespace between collapsed",
			input: "<p>Hello</p><br>\n<br>\n<br><p>World</p>",
			want:  "<p>Hello</p><br/>\n<br/>\n<p>World</p>",
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

func TestSanitizeHTML(t *testing.T) {
	t.Parallel()
	sanitize := SanitizeHTML()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "allows safe elements",
			input: "<p><strong>Hello</strong> <em>World</em></p>",
			want:  "<p><strong>Hello</strong> <em>World</em></p>",
		},
		{
			name:  "strips script tags",
			input: "<p>Hello</p><script>alert('xss')</script><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "strips style tags",
			input: "<p>Hello</p><style>body{color:red}</style><p>World</p>",
			want:  "<p>Hello</p><p>World</p>",
		},
		{
			name:  "strips onclick attributes",
			input: `<p onclick="alert('xss')">Hello</p>`,
			want:  "<p>Hello</p>",
		},
		{
			name:  "allows href on anchors",
			input: `<a href="https://example.com">Link</a>`,
			want:  `<a href="https://example.com" rel="nofollow noreferrer noopener" target="_blank">Link</a>`,
		},
		{
			name:  "allows lists",
			input: "<ul><li>Item 1</li><li>Item 2</li></ul>",
			want:  "<ul><li>Item 1</li><li>Item 2</li></ul>",
		},
		{
			name:  "allows tables",
			input: "<table><tr><td>Cell</td></tr></table>",
			want:  "<table><tr><td>Cell</td></tr></table>",
		},
		{
			name:  "strips images to prevent hotlinking",
			input: "<p>Text<img src=\"image.jpg\">more</p>",
			want:  "<p>Textmore</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := sanitize([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}
