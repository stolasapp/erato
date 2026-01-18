package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterParams_QueryString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params FilterParams
		want   string
	}{
		{
			name:   "empty params",
			params: FilterParams{},
			want:   "",
		},
		{
			name: "type filter only",
			params: FilterParams{
				Filters: Filters{TypeFilter: "story"},
			},
			want: "type=story",
		},
		{
			name: "type all is omitted",
			params: FilterParams{
				Filters: Filters{TypeFilter: "all"},
			},
			want: "",
		},
		{
			name: "unread filter",
			params: FilterParams{
				Filters: Filters{OnlyUnread: true},
			},
			want: "unread=true",
		},
		{
			name: "starred filter",
			params: FilterParams{
				Filters: Filters{OnlyStarred: true},
			},
			want: "starred=true",
		},
		{
			name: "hidden filter",
			params: FilterParams{
				Filters: Filters{ShowHidden: true},
			},
			want: "hidden=true",
		},
		{
			name: "multiple filters",
			params: FilterParams{
				Filters: Filters{
					TypeFilter:  "anthology",
					OnlyUnread:  true,
					OnlyStarred: true,
				},
			},
			want: "starred=true&type=anthology&unread=true",
		},
		{
			name: "page token only",
			params: FilterParams{
				Page: "abc123",
			},
			want: "page=abc123",
		},
		{
			name: "filters and pagination combined",
			params: FilterParams{
				Filters: Filters{
					TypeFilter: "story",
					OnlyUnread: true,
				},
				Page: "xyz789",
			},
			want: "page=xyz789&type=story&unread=true",
		},
		{
			name: "with parent state",
			params: FilterParams{
				Filters: Filters{TypeFilter: "story"},
				Parent:  "hidden=true",
			},
			want: "pf=hidden%3Dtrue&type=story",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.params.QueryString()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterParams_BuildURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  FilterParams
		baseURL string
		want    string
	}{
		{
			name:    "no params",
			params:  FilterParams{},
			baseURL: "/category",
			want:    "/category",
		},
		{
			name: "with params",
			params: FilterParams{
				Filters: Filters{TypeFilter: "story"},
			},
			baseURL: "/category",
			want:    "/category?type=story",
		},
		{
			name: "with pagination",
			params: FilterParams{
				Page: "abc123",
			},
			baseURL: "/my-category",
			want:    "/my-category?page=abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.params.BuildURL(tt.baseURL)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterParams_WithType(t *testing.T) {
	t.Parallel()
	original := FilterParams{
		Filters: Filters{TypeFilter: "story", OnlyUnread: true},
		Page:    "abc123",
	}

	result := original.WithType("anthology")

	assert.Equal(t, "anthology", result.TypeFilter)
	assert.True(t, result.OnlyUnread, "other filters should be preserved")
	assert.Empty(t, result.Page, "pagination should be reset")

	// Original should be unchanged
	assert.Equal(t, "story", original.TypeFilter)
	assert.Equal(t, "abc123", original.Page)
}

func TestFilterParams_WithoutPagination(t *testing.T) {
	t.Parallel()
	original := FilterParams{
		Filters: Filters{TypeFilter: "story", OnlyUnread: true},
		Page:    "abc123",
	}

	result := original.WithoutPagination()

	assert.Equal(t, "story", result.TypeFilter, "filters should be preserved")
	assert.True(t, result.OnlyUnread, "filters should be preserved")
	assert.Empty(t, result.Page, "page should be reset")
}

func TestFilterParams_WithNextPage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		initial   FilterParams
		nextToken string
		wantPage  string
	}{
		{
			name:      "from first page",
			initial:   FilterParams{},
			nextToken: "page2token",
			wantPage:  "page2token",
		},
		{
			name: "from second page",
			initial: FilterParams{
				Page: "page2token",
			},
			nextToken: "page3token",
			wantPage:  "page3token",
		},
		{
			name: "preserves filters",
			initial: FilterParams{
				Filters: Filters{TypeFilter: "story", OnlyUnread: true},
				Page:    "page2token",
			},
			nextToken: "page3token",
			wantPage:  "page3token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := test.initial.WithNextPage(test.nextToken)
			assert.Equal(t, test.wantPage, result.Page)
			// Filters should be preserved
			assert.Equal(t, test.initial.TypeFilter, result.TypeFilter)
			assert.Equal(t, test.initial.OnlyUnread, result.OnlyUnread)
		})
	}
}

func TestFilterParams_ForChild(t *testing.T) {
	t.Parallel()
	parent := FilterParams{
		Filters: Filters{TypeFilter: "story", OnlyUnread: true},
		Page:    "abc123",
	}

	child := parent.ForChild()

	assert.Empty(t, child.TypeFilter, "child should not inherit filters")
	assert.Empty(t, child.Page, "child should not inherit pagination")
	assert.NotEmpty(t, child.Parent, "child should have parent state")
	assert.Contains(t, child.Parent, "type=story")
	assert.Contains(t, child.Parent, "unread=true")
}

func TestFilterParams_ParentFilters(t *testing.T) {
	t.Parallel()
	child := FilterParams{
		Parent: "type=story&unread=true&page=abc123",
	}

	parent := child.ParentFilters()

	assert.Equal(t, "story", parent.TypeFilter)
	assert.True(t, parent.OnlyUnread)
	assert.Equal(t, "abc123", parent.Page)
}

func TestParseQueryString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		qs   string
		want FilterParams
	}{
		{
			name: "empty string",
			qs:   "",
			want: FilterParams{},
		},
		{
			name: "type filter",
			qs:   "type=story",
			want: FilterParams{
				Filters: Filters{TypeFilter: "story"},
			},
		},
		{
			name: "all filters and pagination",
			qs:   "type=anthology&unread=true&starred=true&hidden=true&page=abc123",
			want: FilterParams{
				Filters: Filters{
					TypeFilter:  "anthology",
					OnlyUnread:  true,
					OnlyStarred: true,
					ShowHidden:  true,
				},
				Page: "abc123",
			},
		},
		{
			name: "with parent",
			qs:   "type=story&pf=hidden%3Dtrue",
			want: FilterParams{
				Filters: Filters{TypeFilter: "story"},
				Parent:  "hidden=true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseQueryString(tt.qs)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterParams_RoundTrip(t *testing.T) {
	t.Parallel()
	// Test that navigating forward works correctly
	start := FilterParams{
		Filters: Filters{TypeFilter: "story", OnlyUnread: true},
	}

	// Navigate to page 2
	page2 := start.WithNextPage("token2")
	assert.Equal(t, "token2", page2.Page)

	// Navigate to page 3
	page3 := page2.WithNextPage("token3")
	assert.Equal(t, "token3", page3.Page)

	// Filters should be preserved throughout
	assert.Equal(t, "story", page3.TypeFilter)
	assert.True(t, page3.OnlyUnread)
}

func TestFilterParams_HierarchicalState(t *testing.T) {
	t.Parallel()
	// Test hierarchical filter state (archive -> category -> entry)
	const typeStory = "story"

	// Archive page with hidden filter
	archive := FilterParams{
		Filters: Filters{ShowHidden: true},
	}

	// Navigate to category, store archive state as parent
	category := archive.ForChild()
	category.TypeFilter = typeStory
	category.OnlyUnread = true

	// Verify category has parent state
	assert.Equal(t, typeStory, category.TypeFilter)
	assert.True(t, category.OnlyUnread)
	assert.Contains(t, category.Parent, "hidden=true")

	// Navigate to entry, store category state as parent
	entry := category.ForChild()

	// Entry should have no filters but remember category's state
	assert.Empty(t, entry.TypeFilter)
	assert.Contains(t, entry.Parent, "type=story")
	assert.Contains(t, entry.Parent, "unread=true")

	// Parse entry's parent to get category state
	categoryFromEntry := entry.ParentFilters()
	assert.Equal(t, typeStory, categoryFromEntry.TypeFilter)
	assert.True(t, categoryFromEntry.OnlyUnread)

	// Category's parent contains archive state
	assert.Contains(t, categoryFromEntry.Parent, "hidden=true")

	// Parse to get archive state
	archiveFromCategory := categoryFromEntry.ParentFilters()
	assert.True(t, archiveFromCategory.ShowHidden)
}
