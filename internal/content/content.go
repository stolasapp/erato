// Package content contains transformers to sanitize and render archive content.
package content

import (
	"fmt"
	"mime"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

var (
	// Individual transformers.
	markdownToHTML  = MarkdownToHTML()
	htmlToMarkdown  = HTMLToMarkdown()
	sanitizeHTML    = SanitizeHTML()
	extractHTMLBody = ExtractHTMLBody()
	normalizeNBSP   = NormalizeNBSP()
	scrubHTML       = ScrubHTML()
	scrubText       = ScrubTextDocument()

	// Pre-composed pipelines. UTF8 conversion is handled separately in
	// Transform since it depends on the input charset.
	textToMarkdownPipeline = scrubText
	textToHTMLPipeline     = Chain(scrubText, markdownToHTML, sanitizeHTML)
	htmlToMarkdownPipeline = Chain(
		normalizeNBSP, extractHTMLBody, sanitizeHTML, scrubHTML, htmlToMarkdown,
	)
	htmlToHTMLPipeline = Chain(
		normalizeNBSP, extractHTMLBody, sanitizeHTML, scrubHTML,
	)
)

// Transform converts the input to the desired output mime (media) type.
func Transform(
	inputContentType string,
	outputMimeType eratov1.ReadEntryRequest_MimeType,
	input []byte,
) ([]byte, error) {
	mimeType, _, err := mime.ParseMediaType(inputContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content mime type %q: %w", inputContentType, err)
	}

	// Convert to UTF8 first (charset depends on input)
	input, err = UTF8Transformer(inputContentType)(input)
	if err != nil {
		return nil, err
	}

	// Apply the appropriate pre-composed pipeline
	const plainText = "text/plain"
	switch {
	case mimeType == plainText && outputMimeType == eratov1.ReadEntryRequest_MARKDOWN:
		return textToMarkdownPipeline(input)
	case mimeType == plainText && outputMimeType == eratov1.ReadEntryRequest_HTML:
		return textToHTMLPipeline(input)
	case outputMimeType == eratov1.ReadEntryRequest_MARKDOWN:
		return htmlToMarkdownPipeline(input)
	default:
		return htmlToHTMLPipeline(input)
	}
}
