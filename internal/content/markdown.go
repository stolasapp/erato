package content

import (
	"bytes"
	"fmt"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// HTMLToMarkdown converts an HTML input into CommonMark-compatible Markdown.
func HTMLToMarkdown() TransformerFunc {
	conv := converter.NewConverter(
		converter.WithEscapeMode(converter.EscapeModeSmart),
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(
				commonmark.WithEmDelimiter("_"),
				commonmark.WithHorizontalRule("---"),
				commonmark.WithLinkEmptyContentBehavior(commonmark.LinkBehaviorSkip),
				commonmark.WithLinkEmptyHrefBehavior(commonmark.LinkBehaviorSkip),
			),
			strikethrough.NewStrikethroughPlugin(),
			table.NewTablePlugin(),
		),
	)

	return func(input []byte) ([]byte, error) {
		return conv.ConvertReader(bytes.NewReader(input))
	}
}

// MarkdownToHTML converts a CommonMark Markdown input into HTML. Note that the
// produced HTML is _not_ sanitized.
func MarkdownToHTML() TransformerFunc {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.Linkify,
			extension.Table,
			extension.Strikethrough,
			extension.Typographer,
			extension.CJK,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return func(input []byte) ([]byte, error) {
		output := &bytes.Buffer{}
		if err := markdown.Convert(input, output); err != nil {
			return nil, fmt.Errorf("failed to convert markdown to HTML: %w", err)
		}
		return output.Bytes(), nil
	}
}
