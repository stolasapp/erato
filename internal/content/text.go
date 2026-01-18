package content

import (
	"bytes"
	"regexp"
)

var (
	// Authors sometimes only indicate the start of new paragraphs by
	// indentation and not a blank line between them. This confuses CommonMark
	// into thinking they are the same paragraph. Matches newline followed by
	// either 2+ spaces or a tab (plus any additional whitespace).
	addParagraphWhitespace = regexp.MustCompile(`\n(\s{2}|\t)\s*`)

	// Authors sometimes indent the first line of paragraphs. This confuses
	// CommonMark into thinking these are preformatted text.
	removeIndenting = regexp.MustCompile(`(?m)^[ \t]+`)

	// Authors sometimes use repeated characters as decorative horizontal rules.
	// This catches common patterns (*, -, _, ~, `, =, #, .) on their own line
	// that might confuse CommonMark into thinking they're code fences or
	// headings. Only matches when the pattern is alone on a line.
	decorativeHR = regexp.MustCompile(`(?m)^\s*([-=.*_~\x60#] ?){3,}\s*$`)

	// Trailing whitespace on lines can cause issues and is never intentional.
	trailingWhitespace = regexp.MustCompile(`(?m)[ \t]+$`)
)

// ScrubTextDocument cleans up UTF-8 text into something as compatible as
// possible with CommonMark.
func ScrubTextDocument() TransformerFunc {
	return func(input []byte) ([]byte, error) {
		// Normalize to Unix line endings first
		input = bytes.ReplaceAll(input, []byte("\r\n"), []byte("\n"))
		input = bytes.ReplaceAll(input, []byte("\r"), []byte("\n"))

		input = addParagraphWhitespace.ReplaceAll(input, []byte("\n\n"))
		input = removeIndenting.ReplaceAll(input, nil)
		input = decorativeHR.ReplaceAll(input, []byte("---"))
		input = trailingWhitespace.ReplaceAll(input, nil)
		return input, nil
	}
}
