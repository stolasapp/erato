package content

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// ExtractHTMLBody extracts just the body content from a full HTML document.
// If no body tag exists, returns the input unchanged.
func ExtractHTMLBody() TransformerFunc {
	return func(input []byte) ([]byte, error) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(input))
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML document: %w", err)
		}
		body := doc.Find("body")
		if body.Length() == 0 {
			return input, nil
		}
		innerHTML, err := body.Html()
		if err != nil {
			return nil, fmt.Errorf("failed to extract HTML body: %w", err)
		}
		return []byte(innerHTML), nil
	}
}

// ScrubHTML cleans up common HTML authoring issues like excessive <br> tags
// and empty elements. Should be applied after sanitization.
func ScrubHTML() TransformerFunc {
	return func(input []byte) ([]byte, error) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(input))
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML: %w", err)
		}
		body := doc.Find("body")
		if body.Length() == 0 {
			body = doc.Selection
		}
		scrubHTMLSelection(body)
		out, err := body.Html()
		if err != nil {
			return nil, fmt.Errorf("failed to render scrubbed HTML: %w", err)
		}
		return []byte(out), nil
	}
}

const (
	// maxConsecutiveBRs is the maximum number of consecutive <br> elements
	// allowed before collapsing occurs.
	maxConsecutiveBRs = 2
)

var (
	// nbspPattern matches both the HTML entity &nbsp; (case insensitive) and
	// the actual unicode non-breaking space character (U+00A0).
	nbspPattern = regexp.MustCompile("(?i)&nbsp;|\xc2\xa0")

	// detailsOpenAttr matches the valid values for the details element's open
	// attribute (empty string or "open", case insensitive).
	detailsOpenAttr = regexp.MustCompile(`(?i)^(|open)$`)

	// emailHeadersClass matches only the "email-headers" class value.
	emailHeadersClass = regexp.MustCompile(`^email-headers$`)
)

// NormalizeNBSP replaces non-breaking space entities and characters with
// regular spaces. Operates on raw input before HTML parsing.
func NormalizeNBSP() TransformerFunc {
	return func(input []byte) ([]byte, error) {
		return nbspPattern.ReplaceAll(input, []byte{' '}), nil
	}
}

// SanitizeHTML applies sanitization rules to HTML input, stripping unsupported
// tags and attributes.
func SanitizeHTML() TransformerFunc {
	htmlSanitizer := sanitizer()
	return func(input []byte) ([]byte, error) {
		return htmlSanitizer.SanitizeBytes(input), nil
	}
}

// sanitizer is a modification of [bluemonday.UGCPolicy].
// Differences:
//
//   - Target _blank and noreferrer for links
//   - No figure/image elements (to avoid hot-linking)
//   - No map/area elements
//   - No meter/progress elements
func sanitizer() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()

	policy.AllowStandardAttributes()

	policy.AllowStandardURLs()
	policy.RequireNoReferrerOnLinks(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)

	policy.AllowElements(
		"abbr",
		"acronym",
		"article",
		"aside",
		"b",
		"bdi",
		"bdo",
		"br",
		"cite",
		"code",
		"dfn",
		"div",
		"em",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"hgroup",
		"hr",
		"i",
		"mark",
		"p",
		"pre",
		"rp",
		"rt",
		"ruby",
		"s",
		"samp",
		"section",
		"small",
		"strike",
		"strong",
		"sub",
		"summary",
		"sup",
		"tt",
		"u",
		"var",
		"wbr",
	)

	policy.AllowAttrs("open").
		Matching(detailsOpenAttr).
		OnElements("details")
	policy.AllowAttrs("class").
		Matching(emailHeadersClass).
		OnElements("details")

	policy.AllowAttrs("cite").
		OnElements(
			"blockquote",
			"q",
		)
	policy.AllowAttrs("cite").
		Matching(bluemonday.Paragraph).
		OnElements(
			"del",
			"ins",
		)

	policy.AllowAttrs("href").
		OnElements("a")

	policy.AllowAttrs("datetime").
		Matching(bluemonday.ISO8601).
		OnElements(
			"del",
			"ins",
			"time",
		)

	policy.AllowLists()
	policy.AllowTables()

	return policy
}

// scrubHTMLSelection applies heuristics to clean up common HTML authoring
// issues. Operations are applied in order: empty inline removal, spacer block
// removal, empty block replacement, then excessive BR collapsing.
func scrubHTMLSelection(sel *goquery.Selection) {
	removeEmptyInlineElements(sel)
	removeSpacerBlocks(sel)
	replaceEmptyBlocksWithBR(sel)
	collapseExcessiveBRs(sel)
}

var (
	inlineElements = []string{
		"a", "abbr", "acronym", "b", "bdi", "bdo", "cite", "code", "dfn",
		"em", "i", "mark", "q", "rp", "rt", "ruby", "s", "samp", "small",
		"span", "strike", "strong", "sub", "sup", "tt", "u", "var",
	}
	inlineSelector = strings.Join(inlineElements, ", ")

	blockElements = []string{
		"address", "article", "aside", "blockquote", "div", "footer",
		"header", "hgroup", "main", "nav", "p", "section",
	}
	blockSelector = strings.Join(blockElements, ", ")
)

// removeEmptyInlineElements removes inline elements that have no text content
// and no meaningful children. This cleans up markup like <em></em> or
// <span>   </span> that authors sometimes leave behind.
func removeEmptyInlineElements(sel *goquery.Selection) {
	// Process repeatedly since removing an element may expose its parent
	// as newly empty
	for {
		removed := false
		sel.Find(inlineSelector).Each(func(_ int, el *goquery.Selection) {
			if isEffectivelyEmpty(el) {
				el.Remove()
				removed = true
			}
		})
		if !removed {
			break
		}
	}
}

// removeSpacerBlocks removes block elements that contain only <br> elements
// and whitespace. These are commonly used to force extra vertical spacing
// (e.g., <p><br></p>) and should be stripped entirely.
func removeSpacerBlocks(sel *goquery.Selection) {
	sel.Find(blockSelector).Each(func(_ int, el *goquery.Selection) {
		if isSpacerBlock(el) {
			el.Remove()
		}
	})
}

// isSpacerBlock returns true if the element contains only <br> elements and
// whitespace text nodes. Such elements are used to force extra vertical spacing
// and serve no semantic purpose.
func isSpacerBlock(selection *goquery.Selection) bool {
	node := selection.Get(0)
	if node == nil {
		return false
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case html.TextNode:
			if strings.TrimSpace(child.Data) != "" {
				return false
			}
		case html.ElementNode:
			if child.Data != "br" {
				return false
			}
		default:
			return false
		}
	}
	// Must have at least one <br> to be a spacer (otherwise it's just empty)
	return selection.Find("br").Length() > 0
}

// replaceEmptyBlocksWithBR replaces empty block elements with a <br> to
// preserve the author's intended vertical spacing.
func replaceEmptyBlocksWithBR(sel *goquery.Selection) {
	sel.Find(blockSelector).Each(func(_ int, el *goquery.Selection) {
		if isEffectivelyEmpty(el) {
			el.ReplaceWithHtml("<br>")
		}
	})
}

// isEffectivelyEmpty returns true if the element has no meaningful content:
// no text (ignoring whitespace) and no element children.
func isEffectivelyEmpty(el *goquery.Selection) bool {
	return strings.TrimSpace(el.Text()) == "" && el.Children().Length() == 0
}

// collapseExcessiveBRs finds runs of 3 or more consecutive <br> elements and
// reduces them to 2. Authors sometimes abuse <br> tags for vertical spacing
// instead of using appropriate block elements or CSS.
func collapseExcessiveBRs(sel *goquery.Selection) {
	sel.Find("br").Each(func(_ int, br *goquery.Selection) {
		node := br.Get(0)
		// Check if this BR was already removed as part of an earlier run
		if node.Parent == nil {
			return
		}

		// Walk siblings, counting BRs and removing any beyond the first 2
		count := 1
		for sib := node.NextSibling; sib != nil; {
			next := sib.NextSibling // grab next before potential removal
			if sib.Type == html.TextNode && strings.TrimSpace(sib.Data) == "" {
				sib = next
				continue
			}
			if sib.Type == html.ElementNode && sib.Data == "br" {
				count++
				if count > maxConsecutiveBRs {
					sib.Parent.RemoveChild(sib)
				}
				sib = next
				continue
			}
			break
		}
	})
}
