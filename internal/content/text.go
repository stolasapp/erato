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
		input = stripIsolatedDashPrefixes(input)
		return input, nil
	}
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
