package content

import (
	"bytes"
	"html"
	"regexp"
	"strings"
)

var (
	// Authors sometimes only indicate the start of new paragraphs by
	// indentation and not a blank line between them. This confuses CommonMark
	// into thinking they are the same paragraph. Matches one or more newlines
	// followed by a small indent (2-4 spaces or a tab) - normalizes to single
	// blank line without the indent.
	addParagraphWhitespace = regexp.MustCompile(`\n+( {2,4}|\t)\s*`)

	// Authors sometimes try to center text by adding lots of leading whitespace.
	// This should be stripped without adding paragraph breaks (unlike small indents).
	// Matches 5+ leading spaces at the start of a line.
	stripCenteringWhitespace = regexp.MustCompile(`(?m)^ {5,}`)

	// Authors sometimes indent the first line of paragraphs. This confuses
	// CommonMark into thinking these are preformatted text.
	removeIndenting = regexp.MustCompile(`(?m)^[ \t]+`)

	// Authors sometimes use repeated characters as decorative horizontal rules.
	// This catches common patterns (*, -, _, ~, `, =, #, ., +) on their own line
	// that might confuse CommonMark into thinking they're code fences or
	// headings. Only matches when the pattern is alone on a line.
	decorativeHR = regexp.MustCompile(`(?m)^\s*([-=.*_~\x60#+] ?){3,}\s*$`)

	// sandwichHeaderDash converts "sandwich" headers using dashes to ATX h2.
	// Matches 3+ dashes on lines above and below the header text.
	sandwichHeaderDash = regexp.MustCompile(`(?m)^-{3,}\n([^\n]+)\n-{3,}$`)

	// sandwichHeaderEquals converts "sandwich" headers using equals to ATX h1.
	// Matches 3+ equals on lines above and below the header text.
	sandwichHeaderEquals = regexp.MustCompile(`(?m)^={3,}\n([^\n]+)\n={3,}$`)

	// Trailing whitespace on lines can cause issues and is never intentional.
	trailingWhitespace = regexp.MustCompile(`(?m)[ \t]+$`)

	// emailHeaderBlock matches a block of email headers at the start of a document.
	// Captures: (1) header lines, (2) optional --- separator.
	emailHeaderBlock = regexp.MustCompile(`^((?:[A-Za-z][A-Za-z0-9-]*:[^\n]*\n)+)(---\n)?`)
)

// ScrubTextDocument cleans up UTF-8 text into something as compatible as
// possible with CommonMark.
func ScrubTextDocument() TransformerFunc {
	return func(input []byte) ([]byte, error) {
		// Normalize to Unix line endings first
		input = bytes.ReplaceAll(input, []byte("\r\n"), []byte("\n"))
		input = bytes.ReplaceAll(input, []byte("\r"), []byte("\n"))

		input = wrapEmailHeaders(input)
		input = stripCenteringWhitespace.ReplaceAll(input, nil)
		input = addParagraphWhitespace.ReplaceAll(input, []byte("\n\n"))
		input = removeIndenting.ReplaceAll(input, nil)
		input = sandwichHeaderEquals.ReplaceAll(input, []byte("# $1"))
		input = sandwichHeaderDash.ReplaceAll(input, []byte("## $1"))
		input = decorativeHR.ReplaceAll(input, []byte("***"))
		input = trailingWhitespace.ReplaceAll(input, nil)
		input = stripIsolatedDashPrefixes(input)
		return input, nil
	}
}

// wrapEmailHeaders detects email headers at the start of a document and wraps
// them in a collapsed <details> element to de-emphasize them.
func wrapEmailHeaders(input []byte) []byte {
	// Trim leading whitespace to handle documents that start with blank lines
	trimmed := bytes.TrimLeft(input, " \t\n\r")
	match := emailHeaderBlock.FindSubmatch(trimmed)
	if match == nil {
		return input
	}

	headers := match[1] // The header lines (without ---)
	fullMatch := match[0]

	// HTML-escape the headers to handle <email@address.com> patterns,
	// then convert newlines to <br> for proper line display in HTML.
	escapedHeaders := html.EscapeString(string(bytes.TrimRight(headers, "\n")))
	escapedHeaders = strings.ReplaceAll(escapedHeaders, "\n", "<br>\n")

	var out bytes.Buffer
	out.WriteString("<details class=\"email-headers\">\n<summary>Email headers</summary>\n")
	out.WriteString(escapedHeaders)
	out.WriteString("\n</details>\n\n")

	// Append the rest of the document, trimming leading newlines
	// Note: fullMatch is relative to trimmed, not input
	rest := bytes.TrimLeft(trimmed[len(fullMatch):], "\n")
	out.Write(rest)

	return out.Bytes()
}

// stripIsolatedDashPrefixes removes leading dash prefixes from lines that appear
// to be dialog markers rather than list items. Handles both "- text" and "-text"
// patterns. A dash-prefixed line is considered isolated (and thus a dialog
// marker) if it's not part of a consecutive sequence of dash-prefixed lines
// (ignoring blank lines between them).
func stripIsolatedDashPrefixes(input []byte) []byte {
	lines := bytes.Split(input, []byte("\n"))

	for idx, line := range lines {
		stripLen := dialogDashPrefixLen(line)
		if stripLen == 0 {
			continue
		}

		// Check if isolated (no adjacent dash lines, skipping blanks)
		hasAdjacent := false
		for j := idx - 1; j >= 0 && !hasAdjacent; j-- {
			if len(bytes.TrimSpace(lines[j])) == 0 {
				continue
			}
			hasAdjacent = dialogDashPrefixLen(lines[j]) > 0
			break
		}
		for j := idx + 1; j < len(lines) && !hasAdjacent; j++ {
			if len(bytes.TrimSpace(lines[j])) == 0 {
				continue
			}
			hasAdjacent = dialogDashPrefixLen(lines[j]) > 0
			break
		}

		if !hasAdjacent {
			lines[idx] = line[stripLen:]
		}
	}

	return bytes.Join(lines, []byte("\n"))
}

// Dialog dash prefix lengths.
const (
	dashSpacePrefixLen = 2 // "- "
	dashOnlyPrefixLen  = 1 // "-"
)

// dialogDashPrefixLen returns the length of a dialog dash prefix if present.
// Returns 2 for "- " (dash-space), 1 for "-X" where X is not a dash or space,
// and 0 if no dialog prefix is found.
func dialogDashPrefixLen(line []byte) int {
	if len(line) < dashSpacePrefixLen || line[0] != '-' {
		return 0
	}
	switch line[1] {
	case ' ':
		return dashSpacePrefixLen // "- text"
	case '-':
		return 0 // "--" is not a dialog marker
	default:
		return dashOnlyPrefixLen // "-text" (e.g., -"dialog")
	}
}
