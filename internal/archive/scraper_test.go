package archive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRowTimestamp(t *testing.T) {
	t.Parallel()

	locale, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	scraper := &Scraper{locale: locale}

	tests := []struct {
		name              string
		rowTimestamp      string
		parentLastUpdated time.Time
		now               time.Time // used to describe the test scenario
		want              time.Time
		wantErr           bool
	}{
		{
			name:              "older format with explicit year",
			rowTimestamp:      "Mar 15 2023",
			parentLastUpdated: time.Date(2024, 1, 1, 0, 0, 0, 0, locale),
			want:              time.Date(2023, 3, 15, 0, 0, 0, 0, locale),
		},
		{
			name:              "older format with single digit day",
			rowTimestamp:      "Jan  5 2022",
			parentLastUpdated: time.Date(2024, 1, 1, 0, 0, 0, 0, locale),
			want:              time.Date(2022, 1, 5, 0, 0, 0, 0, locale),
		},
		{
			name:         "invalid format",
			rowTimestamp: "2023-03-15",
			wantErr:      true,
		},
		{
			name:         "empty string",
			rowTimestamp: "",
			wantErr:      true,
		},
		{
			name:         "malformed recent format",
			rowTimestamp: "Mar 15",
			wantErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := scraper.parseRowTimestamp(test.rowTimestamp, test.parentLastUpdated)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

// TestParseRowTimestampYearInference tests the year inference logic for recent
// timestamps that don't include a year. This is more complex because it depends
// on both the parent's last modified date and the current time.
func TestParseRowTimestampYearInference(t *testing.T) {
	t.Parallel()

	locale, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	scraper := &Scraper{locale: locale}

	// Note: The year inference logic uses time.Now() internally, so these tests
	// verify the behavior based on the current time at test execution.
	// The tests are designed to be stable regardless of when they run.

	now := time.Now().In(locale)
	currentYear := now.Year()

	tests := []struct {
		name              string
		rowTimestamp      string
		parentLastUpdated time.Time
		wantYear          int
	}{
		{
			name:              "recent timestamp same year as parent and now",
			rowTimestamp:      now.Format("Jan _2 15:04"),
			parentLastUpdated: now,
			wantYear:          currentYear,
		},
		{
			name:              "parent in previous year, row month after parent month",
			rowTimestamp:      "Feb  1 12:00",
			parentLastUpdated: time.Date(currentYear-1, 1, 15, 0, 0, 0, 0, locale),
			wantYear:          currentYear - 2, // crossed back another year
		},
		{
			name:              "parent in previous year, row month same or before parent month",
			rowTimestamp:      "Jan  1 12:00",
			parentLastUpdated: time.Date(currentYear-1, 2, 15, 0, 0, 0, 0, locale),
			wantYear:          currentYear - 1, // same year as parent
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := scraper.parseRowTimestamp(test.rowTimestamp, test.parentLastUpdated)
			require.NoError(t, err)
			assert.Equal(t, test.wantYear, got.Year())
		})
	}
}
