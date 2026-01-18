package component

import (
	"log/slog"

	"github.com/stolasapp/erato/internal/slugconv"
)

// EntrySlug converts an entry path to a URL slug, logging any errors.
func EntrySlug(path string) string {
	slug, err := slugconv.FromEntryPath(path)
	if err != nil {
		slog.Error("failed to convert entry path to slug",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
	return slug
}

// ChapterSlug converts a chapter path to a URL slug, logging any errors.
func ChapterSlug(path string) string {
	slug, err := slugconv.FromChapterPath(path)
	if err != nil {
		slog.Error("failed to convert chapter path to slug",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
	return slug
}

// CategorySlug converts a category path to a URL slug, logging any errors.
func CategorySlug(path string) string {
	slug, err := slugconv.FromCategoryPath(path)
	if err != nil {
		slog.Error("failed to convert category path to slug",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
	return slug
}
