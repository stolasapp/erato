package slugconv

import (
	"path"
	"strings"
	"unicode"
)

// ToTitle converts a slug into a rough approximation of title case. It extracts
// the base name, strips any .html suffix, replaces hyphens with spaces, and
// capitalizes the first letter of each word.
func ToTitle(slug string) string {
	name := path.Base(slug)
	name = strings.TrimSuffix(name, ".html")
	name = strings.ReplaceAll(name, "-", " ")
	return titleCase(name)
}

// titleCase capitalizes the first rune of each space-separated word.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
