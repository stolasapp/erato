package component

import (
	"net/url"
)

const boolTrue = "true"

// FilterParams holds the current filter and pagination state for URL building.
// It embeds Filters for backward compatibility with existing code.
type FilterParams struct {
	Filters

	Page   string // Current page token (empty for first page)
	Parent string // Parent page's complete query string (for hierarchical navigation)
}

// QueryString returns the query string portion of the URL (without leading ?).
func (f FilterParams) QueryString() string {
	params := url.Values{}

	// Filter params
	if f.TypeFilter != "" && f.TypeFilter != "all" {
		params.Set("type", f.TypeFilter)
	}
	if f.OnlyUnread {
		params.Set("unread", "true")
	}
	if f.OnlyStarred {
		params.Set("starred", "true")
	}
	if f.ShowHidden {
		params.Set("hidden", "true")
	}

	// Pagination params
	if f.Page != "" {
		params.Set("page", f.Page)
	}

	// Parent state (for hierarchical navigation)
	if f.Parent != "" {
		params.Set("pf", f.Parent)
	}

	return params.Encode()
}

// BuildURL constructs a full URL with the base path and query parameters.
func (f FilterParams) BuildURL(baseURL string) string {
	qs := f.QueryString()
	if qs == "" {
		return baseURL
	}
	return baseURL + "?" + qs
}

// WithType returns a copy with the type filter changed.
// This resets pagination since the result set changes.
func (f FilterParams) WithType(typeVal string) FilterParams {
	f.TypeFilter = typeVal
	return f.WithoutPagination()
}

// WithUnread returns a copy with the unread filter toggled.
// This resets pagination since the result set changes.
func (f FilterParams) WithUnread(unread bool) FilterParams {
	f.OnlyUnread = unread
	return f.WithoutPagination()
}

// WithStarred returns a copy with the starred filter toggled.
// This resets pagination since the result set changes.
func (f FilterParams) WithStarred(starred bool) FilterParams {
	f.OnlyStarred = starred
	return f.WithoutPagination()
}

// WithHidden returns a copy with the hidden filter toggled.
// This resets pagination since the result set changes.
func (f FilterParams) WithHidden(hidden bool) FilterParams {
	f.ShowHidden = hidden
	return f.WithoutPagination()
}

// WithoutPagination returns a copy with pagination reset to the first page.
// Use this when changing filters.
func (f FilterParams) WithoutPagination() FilterParams {
	f.Page = ""
	return f
}

// WithNextPage returns a copy for navigating to the next page.
func (f FilterParams) WithNextPage(nextToken string) FilterParams {
	f.Page = nextToken
	return f
}

// ForChild returns FilterParams for a child page, storing current state as parent.
// The child page has no filters of its own but remembers the parent's state.
func (f FilterParams) ForChild() FilterParams {
	return FilterParams{
		Parent: f.QueryString(),
	}
}

// ParentFilters returns the parent's FilterParams parsed from the Parent field.
// Returns an empty FilterParams if there is no parent state.
func (f FilterParams) ParentFilters() FilterParams {
	if f.Parent == "" {
		return FilterParams{}
	}
	return ParseQueryString(f.Parent)
}

// ParseQueryString parses a query string into FilterParams.
func ParseQueryString(qs string) FilterParams {
	values, err := url.ParseQuery(qs)
	if err != nil {
		return FilterParams{}
	}
	return FilterParams{
		Filters: Filters{
			TypeFilter:  values.Get("type"),
			OnlyUnread:  values.Get("unread") == boolTrue,
			OnlyStarred: values.Get("starred") == boolTrue,
			ShowHidden:  values.Get("hidden") == boolTrue,
		},
		Page:   values.Get("page"),
		Parent: values.Get("pf"),
	}
}
