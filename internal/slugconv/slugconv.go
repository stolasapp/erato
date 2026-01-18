// Package slugconv handles conversions between resource slugs and paths
package slugconv

import (
	"fmt"
	"path"
	"regexp"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

const (
	// ErrInvalidPath is returned for malformed path strings.
	ErrInvalidPath = Error("invalid path")
	// ErrInvalidSlug is returned for malformed slug strings.
	ErrInvalidSlug = Error("invalid slug")
)

const (
	categoryCollection = "categories"
	entryCollection    = "entries"
	chapterCollection  = "chapters"

	resourceIDPattern = `[a-zA-Z0-9]([a-z0-9-.]{0,61}[a-z0-9])?`

	categoryPathPattern = categoryCollection + "/(" + resourceIDPattern + ")"
	categoryPathFormat  = categoryCollection + "/%s"
	categorySlugPattern = "(" + resourceIDPattern + ")"
	categorySlugFormat  = "%s"

	entryPathPattern = categoryPathPattern + "/" + entryCollection + "/(" + resourceIDPattern + ")"
	entryPathFormat  = categoryPathFormat + "/" + entryCollection + "/%s"
	entrySlugPattern = categorySlugPattern + "/(" + resourceIDPattern + ")"
	entrySlugFormat  = categorySlugFormat + "/%s"

	chapterPathPattern = entryPathPattern + "/" + chapterCollection + "/(" + resourceIDPattern + ")"
	chapterPathFormat  = entryPathFormat + "/" + chapterCollection + "/%s"
	chapterSlugPattern = entrySlugPattern + "/(" + resourceIDPattern + ")"
	chapterSlugFormat  = entrySlugFormat + "/%s"

	oneToManyPathSegments = 2
	oneToOnePathSegments  = 1
)

var (
	// Indexes of the path components in the regex matches.
	categoryMatchIndices = []int{1}
	entryMatchIndices    = []int{1, 3}
	chapterMatchIndices  = []int{1, 3, 5}

	// Regex patterns exact-matching a resource path.
	categoryPath = patternMustCompile(categoryPathPattern, false)
	entryPath    = patternMustCompile(entryPathPattern, false)
	chapterPath  = patternMustCompile(chapterPathPattern, false)

	// Regex patterns exact-matching a resource slug.
	categorySlug = patternMustCompile(categorySlugPattern, true)
	entrySlug    = patternMustCompile(entrySlugPattern, true)
	chapterSlug  = patternMustCompile(chapterSlugPattern, false)
)

// Error is an error type for slug/path conversion failures.
type Error string

// Error satisfies [error].
func (e Error) Error() string { return string(e) }

// ToCategoryPath converts a category slug into a path.
func ToCategoryPath(slug string) (path string, err error) {
	return toPath(slug, categorySlug, categoryPathFormat, categoryMatchIndices)
}

// ToEntryPath converts an entry slug into a path.
func ToEntryPath(slug string) (path string, err error) {
	return toPath(slug, entrySlug, entryPathFormat, entryMatchIndices)
}

// ToChapterPath converts a chapter slug into a path.
func ToChapterPath(slug string) (path string, err error) {
	return toPath(slug, chapterSlug, chapterPathFormat, chapterMatchIndices)
}

// FromProto resolves the slug for an arbitrary eratov1 message.
func FromProto(msg interface{ GetPath() string }) (slug string, err error) {
	switch msg := msg.(type) {
	case *eratov1.Category:
		return FromCategoryPath(msg.GetPath())
	case *eratov1.Entry:
		return FromEntryPath(msg.GetPath())
	case *eratov1.Chapter:
		return FromChapterPath(msg.GetPath())
	default:
		return "", fmt.Errorf("unexpected type %T", msg)
	}
}

// FromCategoryPath converts a category path into a slug.
func FromCategoryPath(path string) (slug string, err error) {
	return toSlug(path, categoryPath, categorySlugFormat, categoryMatchIndices)
}

// FromEntryPath converts an entry path into a slug.
func FromEntryPath(path string) (slug string, err error) {
	return toSlug(path, entryPath, entrySlugFormat, entryMatchIndices)
}

// FromChapterPath converts a chapter path into a slug.
func FromChapterPath(path string) (slug string, err error) {
	return toSlug(path, chapterPath, chapterSlugFormat, chapterMatchIndices)
}

// EntryParent converts an entry path into its parent category's path.
func EntryParent(path string) (category string) {
	return toParent(path, oneToManyPathSegments)
}

// ChapterParent converts a chapter path into its parent entry's path.
func ChapterParent(path string) (entry string) {
	return toParent(path, oneToManyPathSegments)
}

func toParent(resourcePath string, depth int) (parent string) {
	parent = resourcePath
	for range depth {
		parent = path.Dir(parent)
	}
	return parent
}

func toPath(slug string, pattern *regexp.Regexp, format string, matchIndices []int) (path string, err error) {
	return convert(slug, pattern, format, ErrInvalidSlug, "slug", matchIndices)
}

func toSlug(path string, pattern *regexp.Regexp, format string, matchIndices []int) (slug string, err error) {
	return convert(path, pattern, format, ErrInvalidPath, "path", matchIndices)
}

func convert(
	from string,
	pattern *regexp.Regexp,
	format string,
	matchErr Error,
	typ string,
	matchIndices []int,
) (to string, err error) {
	matches := pattern.FindStringSubmatch(from)
	if len(matches) == 0 {
		return "", fmt.Errorf("%w: %s %q does not match pattern `%s`", matchErr, typ, from, pattern.String())
	}
	args := make([]any, len(matchIndices))
	for i, index := range matchIndices {
		args[i] = matches[index]
	}
	return fmt.Sprintf(format, args...), nil
}

func patternMustCompile(pattern string, optionalTrailingSlash bool) *regexp.Regexp {
	if optionalTrailingSlash {
		pattern += "/?"
	}
	return regexp.MustCompile("^" + pattern + "$")
}
