package archive

import (
	"context"
	"testing"

	celext "buf.build/go/protovalidate/cel"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

func TestCompileFilter(t *testing.T) {
	t.Parallel()

	// Set up a CEL environment for categories (simplest type to test with)
	base, err := cel.NewEnv(cel.Lib(celext.NewLibrary()))
	require.NoError(t, err)

	env, err := initCELEnv(base, categoriesFieldDesc, categoriesCELType, "categories")
	require.NoError(t, err)

	tests := []struct {
		name    string
		filter  string
		wantErr string
	}{
		{
			name:   "valid boolean filter",
			filter: "true",
		},
		{
			name:   "valid field access filter",
			filter: "this.display_name != ''",
		},
		{
			name:   "valid string contains filter",
			filter: "this.display_name.contains('test')",
		},
		{
			name:    "invalid syntax",
			filter:  "this.title ==",
			wantErr: "failed to compile filter",
		},
		{
			name:    "undefined field",
			filter:  "this.nonexistent_field == 'foo'",
			wantErr: "failed to compile filter",
		},
		{
			name:    "wrong return type - string instead of bool",
			filter:  "this.display_name",
			wantErr: "no matching overload",
		},
		{
			name:    "wrong return type - int instead of bool",
			filter:  "1 + 1",
			wantErr: "no matching overload",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			prog, err := compileFilter(env, categoriesCELType, test.filter)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, prog)
		})
	}
}

func TestCompileFilterEvaluation(t *testing.T) {
	t.Parallel()

	// Set up a CEL environment for categories
	base, err := cel.NewEnv(cel.Lib(celext.NewLibrary()))
	require.NoError(t, err)

	env, err := initCELEnv(base, categoriesFieldDesc, categoriesCELType, "categories")
	require.NoError(t, err)

	// Create test data
	categories := []*eratov1.Category{
		eratov1.Category_builder{Path: "categories/foo", DisplayName: "Foo Category"}.Build(),
		eratov1.Category_builder{Path: "categories/bar", DisplayName: "Bar Category"}.Build(),
		eratov1.Category_builder{Path: "categories/baz", DisplayName: "Baz Test"}.Build(),
	}

	tests := []struct {
		name      string
		filter    string
		wantPaths []string
		wantEmpty bool
	}{
		{
			name:      "filter all - true",
			filter:    "true",
			wantPaths: []string{"categories/foo", "categories/bar", "categories/baz"},
		},
		{
			name:      "filter none - false",
			filter:    "false",
			wantEmpty: true,
		},
		{
			name:      "filter by display_name contains",
			filter:    "this.display_name.contains('Test')",
			wantPaths: []string{"categories/baz"},
		},
		{
			name:      "filter by path prefix",
			filter:    "this.path.startsWith('categories/b')",
			wantPaths: []string{"categories/bar", "categories/baz"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			prog, err := compileFilter(env, categoriesCELType, test.filter)
			require.NoError(t, err)

			val, _, err := prog.ContextEval(context.Background(), map[string]any{
				resultsVar: categories,
			})
			require.NoError(t, err)

			// Extract paths from results
			resultList, ok := val.Value().([]ref.Val)
			require.True(t, ok, "expected []ref.Val, got %T", val.Value())
			gotPaths := make([]string, 0, len(resultList))
			for _, item := range resultList {
				cat, ok := item.Value().(*eratov1.Category)
				require.True(t, ok, "expected *Category, got %T", item.Value())
				gotPaths = append(gotPaths, cat.GetPath())
			}

			if test.wantEmpty {
				assert.Empty(t, gotPaths)
			} else {
				assert.Equal(t, test.wantPaths, gotPaths)
			}
		})
	}
}

func TestNewPaginator(t *testing.T) {
	t.Parallel()

	// Test that NewPaginator successfully initializes all CEL environments
	paginator, err := NewPaginator(nil)
	require.NoError(t, err)
	assert.NotNil(t, paginator.categoriesEnv)
	assert.NotNil(t, paginator.entriesEnv)
	assert.NotNil(t, paginator.chaptersEnv)
	assert.NotNil(t, paginator.usersEnv)
}
