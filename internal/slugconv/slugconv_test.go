package slugconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

func TestToCategoryPath(t *testing.T) {
	t.Parallel()
	assertConverts(t,
		ToCategoryPath,
		"foo-bar",
		"categories/foo-bar",
	)
	assertConvertError(t,
		ToCategoryPath,
		"foo-bar/baz",
		ErrInvalidSlug,
	)
}

func TestToEntryPath(t *testing.T) {
	t.Parallel()
	assertConverts(t,
		ToEntryPath,
		"foo-bar/fizz-buzz",
		"categories/foo-bar/entries/fizz-buzz",
	)
	assertConvertError(t,
		ToEntryPath,
		"foo-bar/fizz-buzz/baz",
		ErrInvalidSlug,
	)
}

func TestToChapterPath(t *testing.T) {
	t.Parallel()
	assertConverts(t,
		ToChapterPath,
		"foo-bar/fizz-buzz/fizz-buzz-1",
		"categories/foo-bar/entries/fizz-buzz/chapters/fizz-buzz-1",
	)
	assertConvertError(t,
		ToChapterPath,
		"foo-bar/fizz-buzz",
		ErrInvalidSlug,
	)
}

func TestFromCategoryPath(t *testing.T) {
	t.Parallel()
	assertConverts(t,
		FromCategoryPath,
		"categories/foo-bar",
		"foo-bar",
	)
	assertConvertError(t,
		FromCategoryPath,
		"categories/foo-bar/baz",
		ErrInvalidPath,
	)
}

func TestFromEntryPath(t *testing.T) {
	t.Parallel()
	assertConverts(t,
		FromEntryPath,
		"categories/foo-bar/entries/fizz-buzz",
		"foo-bar/fizz-buzz",
	)
	assertConvertError(t,
		FromEntryPath,
		"categories/foo-bar/fizz-buzz/baz",
		ErrInvalidPath,
	)
}

func TestFromChapterPath(t *testing.T) {
	t.Parallel()
	assertConverts(t,
		FromChapterPath,
		"categories/foo-bar/entries/fizz-buzz/chapters/fizz-buzz-1",
		"foo-bar/fizz-buzz/fizz-buzz-1",
	)
	assertConvertError(t,
		FromChapterPath,
		"categories/foo-bar/fizz-buzz",
		ErrInvalidPath,
	)
}

func assertConverts(t *testing.T, fn func(string) (string, error), input, expected string) {
	t.Helper()
	actual, err := fn(input)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func assertConvertError(t *testing.T, fn func(string) (string, error), input string, expected error) {
	t.Helper()
	_, err := fn(input)
	require.ErrorIs(t, err, expected)
}

func TestEntryParent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "entry to category",
			path: "categories/foo/entries/bar",
			want: "categories/foo",
		},
		{
			name: "nested path",
			path: "categories/foo-bar/entries/baz-qux",
			want: "categories/foo-bar",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, EntryParent(test.path))
		})
	}
}

func TestChapterParent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "chapter to entry",
			path: "categories/foo/entries/bar/chapters/ch1",
			want: "categories/foo/entries/bar",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, ChapterParent(test.path))
		})
	}
}

func TestFromProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		msg     interface{ GetPath() string }
		want    string
		wantErr bool
	}{
		{
			name: "category",
			msg:  eratov1.Category_builder{Path: "categories/foo"}.Build(),
			want: "foo",
		},
		{
			name: "entry",
			msg:  eratov1.Entry_builder{Path: "categories/foo/entries/bar"}.Build(),
			want: "foo/bar",
		},
		{
			name: "chapter",
			msg:  eratov1.Chapter_builder{Path: "categories/foo/entries/bar/chapters/ch1"}.Build(),
			want: "foo/bar/ch1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := FromProto(test.msg)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.want, got)
			}
		})
	}
}

func TestToTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		slug string
		want string
	}{
		{
			name: "simple slug",
			slug: "hello-world",
			want: "Hello World",
		},
		{
			name: "path with multiple segments",
			slug: "foo/bar/hello-world",
			want: "Hello World",
		},
		{
			name: "strips html extension",
			slug: "my-page.html",
			want: "My Page",
		},
		{
			name: "single word",
			slug: "hello",
			want: "Hello",
		},
		{
			name: "preserves existing caps",
			slug: "mySQL-database",
			want: "MySQL Database",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, ToTitle(test.slug))
		})
	}
}
