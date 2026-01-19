package uitest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stolasapp/erato/internal/app/component"
)

const (
	// defaultTimeout is the default timeout for all browser operations.
	defaultTimeout = 10 * time.Second
	// stableTimeout is the timeout for waiting for page stability.
	stableTimeout = 5 * time.Second
)

// testPage wraps a rod.Page with consistent timeout handling.
type testPage struct {
	*rod.Page

	t *testing.T
}

// el finds a single element with the default timeout.
func (p *testPage) el(selector string) *rod.Element {
	return p.Page.Timeout(defaultTimeout).MustElement(selector)
}

// els finds multiple elements with the default timeout.
func (p *testPage) els(selector string) rod.Elements {
	els, _ := p.Page.Timeout(defaultTimeout).Elements(selector)
	return els
}

// elMaybe finds an element or returns nil if not found.
func (p *testPage) elMaybe(selector string) *rod.Element {
	el, err := p.Page.Timeout(defaultTimeout).Element(selector)
	if err != nil {
		return nil
	}
	return el
}

// click clicks an element found by selector and waits for page stability.
func (p *testPage) click(selector string) {
	p.el(selector).MustClick()
	p.waitStable()
}

// clickAndWaitIdle clicks an element and waits for network requests to complete.
// Use this for HTMX triggers where the response swaps DOM content.
func (p *testPage) clickAndWaitIdle(el *rod.Element) {
	el.MustClick()
	p.waitRequestIdle()
}

// waitStable waits for the page to stabilize after HTMX updates.
func (p *testPage) waitStable() {
	p.Page.Timeout(stableTimeout).MustWaitStable()
}

// waitRequestIdle waits for all network requests to complete.
// Use this instead of waitStable when dealing with HTMX swaps that may
// cause continuous DOM updates (e.g., content pages with dynamic elements).
func (p *testPage) waitRequestIdle() {
	p.Page.Timeout(defaultTimeout).MustWaitRequestIdle()()
}

// waitLoad waits for the page to fully load (for full page navigations).
func (p *testPage) waitLoad() {
	p.Page.Timeout(defaultTimeout).MustWaitLoad()
	p.waitStable()
}

// reload reloads the page and waits for stability.
func (p *testPage) reload() {
	p.Page.Timeout(defaultTimeout).MustReload()
	p.waitStable()
}

// TestUI is the parent test that sets up the browser and server,
// then runs all UI subtests. It skips when running with -short flag.
func TestUI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI tests in short mode")
	}

	// Setup test server
	server := newTestServer()
	t.Cleanup(server.Close)

	// Setup headless browser
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).Headless(true).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { browser.MustClose() })

	// Helper to create a new page for each subtest
	newPage := func(t *testing.T) *testPage {
		t.Helper()
		page := browser.Timeout(defaultTimeout).MustPage(server.URL("/"))
		t.Cleanup(func() {
			_ = page.Close()
		})
		page.Timeout(stableTimeout).MustWaitStable()
		return &testPage{Page: page, t: t}
	}

	// Run subtests serially to avoid browser contention
	t.Run("ArchivePage", func(t *testing.T) {
		testArchivePage(t, newPage)
	})
	t.Run("CategoryPage", func(t *testing.T) {
		testCategoryPage(t, newPage)
	})
	t.Run("AnthologyPage", func(t *testing.T) {
		testAnthologyPage(t, newPage)
	})
	t.Run("StoryPage", func(t *testing.T) {
		testStoryPage(t, newPage)
	})
	t.Run("BreadcrumbNavigation", func(t *testing.T) {
		testBreadcrumbNavigation(t, newPage, server)
	})
	t.Run("FilterToggleNoLayoutShift", func(t *testing.T) {
		testFilterToggleNoLayoutShift(t, newPage)
	})
	t.Run("ArchiveHiddenFilter", func(t *testing.T) {
		testArchiveHiddenFilter(t, newPage)
	})
	t.Run("CategoryTypeFilter", func(t *testing.T) {
		testCategoryTypeFilter(t, newPage)
	})
	t.Run("CategoryBooleanFilters", func(t *testing.T) {
		testCategoryBooleanFilters(t, newPage)
	})
	t.Run("CategoryHiddenFilter", func(t *testing.T) {
		testCategoryHiddenFilter(t, newPage)
	})
	t.Run("AutoView", func(t *testing.T) {
		testAutoView(t, newPage)
	})
	t.Run("UpdatedTimestamp", func(t *testing.T) {
		testUpdatedTimestamp(t, newPage, server)
	})
	t.Run("StarToggle", func(t *testing.T) {
		testStarToggle(t, newPage)
	})
	t.Run("ChapterPage", func(t *testing.T) {
		testChapterPage(t, newPage)
	})
	t.Run("FilterCombination", func(t *testing.T) {
		testFilterCombination(t, newPage)
	})
	t.Run("PaginationNavigation", func(t *testing.T) {
		testPaginationNavigation(t, newPage, server)
	})
	t.Run("BreadcrumbFilterPropagation", func(t *testing.T) {
		testBreadcrumbFilterPropagation(t, newPage, server)
	})
	t.Run("MarkReadFilterPreservation", func(t *testing.T) {
		testMarkReadFilterPreservation(t, newPage, server)
	})
	t.Run("ContentPageToggle", func(t *testing.T) {
		testContentPageToggle(t, newPage)
	})
}

// navigateToCategory clicks the first visible category and waits for navigation.
func navigateToCategory(p *testPage) {
	p.click(SelectorListItem + ":not([data-hidden]) > a")
}

// navigateToStory navigates to a category and clicks the first visible story.
// Returns false if no stories are available.
func navigateToStory(p *testPage) bool {
	navigateToCategory(p)
	stories := p.els(VisibleItemByKind(component.KindStory))
	if len(stories) == 0 {
		return false
	}
	stories[0].Timeout(defaultTimeout).MustElement("a").MustClick()
	p.waitStable()
	return true
}

// navigateToAnthology navigates to a category and clicks the first visible anthology.
// Returns false if no anthologies are available.
func navigateToAnthology(p *testPage) bool {
	navigateToCategory(p)
	anthologies := p.els(VisibleItemByKind(component.KindAnthology))
	if len(anthologies) == 0 {
		return false
	}
	anthologies[0].Timeout(defaultTimeout).MustElement("a").MustClick()
	p.waitStable()
	return true
}

// navigateToChapter navigates to an anthology and clicks the first chapter.
// Returns false if no chapters are available.
func navigateToChapter(p *testPage) bool {
	if !navigateToAnthology(p) {
		return false
	}
	chapters := p.els(SelectorChapterItem)
	if len(chapters) == 0 {
		return false
	}
	chapters[0].Timeout(defaultTimeout).MustElement("a").MustClick()
	p.waitStable()
	return true
}

// findFilterButton finds a filter toggle button by text content.
func findFilterButton(p *testPage, text string) *rod.Element {
	for _, btn := range p.els(SelectorFilters + " > button") {
		if strings.Contains(btn.MustText(), text) {
			return btn
		}
	}
	return nil
}

// findFilterSegment finds a filter segment by exact text.
// Uses a retry mechanism to handle HTMX swap timing.
func findFilterSegment(p *testPage, text string) *rod.Element {
	const maxRetries = 10
	const retryDelay = 100 * time.Millisecond

	segmentSelector := SelectorFilters + " [role='group'] button"
	for range maxRetries {
		// Wait for at least one segment button to exist
		if _, err := p.Page.Timeout(defaultTimeout).Element(segmentSelector); err != nil {
			return nil
		}

		for _, seg := range p.els(segmentSelector) {
			if strings.TrimSpace(seg.MustText()) == text {
				return seg
			}
		}

		time.Sleep(retryDelay)
	}
	return nil
}

// ensureHiddenItem hides the first list item if none are hidden.
func ensureHiddenItem(p *testPage) {
	if len(p.els(SelectorListItem+"[data-hidden]")) == 0 {
		firstItem := p.el(SelectorListItem)
		firstItem.Timeout(defaultTimeout).MustElement(SelectorHideButton).MustClick()
		p.waitStable()
	}
}

// testArchivePage tests the archive (category list) page.
func testArchivePage(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)

	// Verify category list loads
	require.NotNil(t, p.el(SelectorListContainer))
	assert.NotEmpty(t, p.els(SelectorListItem), "expected at least one category")

	// Verify filter bar with hidden filter (but no starred filter)
	require.NotNil(t, p.el(SelectorFilters))
	require.NotNil(t, findFilterButton(p, "Hidden"), "expected Hidden filter button")
	assert.Nil(t, findFilterButton(p, "Starred"), "category list should not have starred filter")
}

// testCategoryPage tests the category (entry list) page.
func testCategoryPage(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)
	navigateToCategory(p)

	// Verify entries load
	assert.NotEmpty(t, p.els(SelectorListItem), "expected at least one entry")

	// Verify type filter segments
	segments := p.els(SelectorFilters + " [role='group'] button")
	assert.Len(t, segments, 3, "expected All, Stories, Anthologies segments")

	segmentTexts := make([]string, len(segments))
	for i, seg := range segments {
		segmentTexts[i] = strings.TrimSpace(seg.MustText())
	}
	assert.Contains(t, segmentTexts, "All")
	assert.Contains(t, segmentTexts, "Stories")
	assert.Contains(t, segmentTexts, "Anthologies")

	// Verify filter buttons
	assert.NotNil(t, findFilterButton(p, "Unread"), "expected Unread filter")
	assert.NotNil(t, findFilterButton(p, "Starred"), "expected Starred filter")
	assert.NotNil(t, findFilterButton(p, "Hidden"), "expected Hidden filter")
}

// testAnthologyPage tests the anthology (chapter list) page.
func testAnthologyPage(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)

	if !navigateToAnthology(p) {
		t.Skip("no anthologies found")
	}

	// Verify chapter list loads with no filter bar
	assert.NotEmpty(t, p.els(SelectorListItem), "anthology should have chapters")
	assert.NotEmpty(t, p.els(SelectorChapterItem), "chapters should have chapter data-kind")
	assert.Empty(t, p.els(SelectorFilters), "anthology page should not have filter bar")
}

// testStoryPage tests navigating to a story and viewing content.
func testStoryPage(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)

	if !navigateToStory(p) {
		t.Skip("no stories found")
	}

	content := p.el("article")
	require.NotNil(t, content)
	assert.NotEmpty(t, content.MustText(), "expected content to have text")
}

// testBreadcrumbNavigation tests breadcrumb navigation.
func testBreadcrumbNavigation(t *testing.T, newPage func(*testing.T) *testPage, server *Server) {
	p := newPage(t)

	// Get category name before navigating
	categoryLink := p.el(SelectorListItem + ":not([data-hidden]) > a")
	categoryName := categoryLink.MustText()
	categoryLink.MustClick()
	p.waitStable()

	// Verify breadcrumb shows category
	breadcrumbs := p.el(SelectorBreadcrumbs)
	require.NotNil(t, breadcrumbs)
	assert.Contains(t, breadcrumbs.MustText(), categoryName)

	// Navigate home via site title
	p.click(SelectorSiteTitle + " a")
	assert.Equal(t, server.URL("/"), p.MustInfo().URL)
}

// testFilterToggleNoLayoutShift tests that filter toggle doesn't shift layout.
func testFilterToggleNoLayoutShift(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)

	hiddenBtn := p.el(SelectorFilters + " > button")
	initialBox := hiddenBtn.MustShape().Box()

	hiddenBtn.MustClick()
	p.waitStable()

	hiddenBtn = p.el(SelectorFilters + " > button")
	afterBox := hiddenBtn.MustShape().Box()

	assert.InDelta(t, initialBox.Y, afterBox.Y, 2.0, "button should not shift vertically")
}

// testArchiveHiddenFilter tests the hidden filter on the archive page.
func testArchiveHiddenFilter(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)

	// Toggle test
	hiddenBtn := findFilterButton(p, "Hidden")
	require.NotNil(t, hiddenBtn)

	ariaPressed := hiddenBtn.MustAttribute("aria-pressed")
	assert.Equal(t, "false", *ariaPressed, "should start inactive")

	hiddenBtn.MustClick()
	p.waitStable()

	hiddenBtn = findFilterButton(p, "Hidden")
	ariaPressed = hiddenBtn.MustAttribute("aria-pressed")
	assert.Equal(t, "true", *ariaPressed, "should be active after click")

	hiddenBtn.MustClick()
	p.waitStable()

	hiddenBtn = findFilterButton(p, "Hidden")
	ariaPressed = hiddenBtn.MustAttribute("aria-pressed")
	assert.Equal(t, "false", *ariaPressed, "should be inactive after second click")

	// Visibility test - ensure we have a hidden item
	ensureHiddenItem(p)

	hiddenItems := p.els(SelectorListItem + "[data-hidden]")
	require.NotEmpty(t, hiddenItems, "should have hidden items")

	// Hidden items should be invisible
	for _, item := range hiddenItems {
		box := item.MustShape().Box()
		assert.LessOrEqual(t, box.Height, 1.0, "hidden item should have minimal height when filter is off")
	}

	// Enable filter
	findFilterButton(p, "Hidden").MustClick()
	p.waitStable()

	// Hidden items should now be visible
	for _, item := range p.els(SelectorListItem + "[data-hidden]") {
		box := item.MustShape().Box()
		assert.Greater(t, box.Height, 1.0, "hidden item should be visible when filter is on")
	}
}

// testCategoryTypeFilter tests the type filter (All/Stories/Anthologies).
func testCategoryTypeFilter(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)
	navigateToCategory(p)

	if len(p.els(SelectorStoryItem)) == 0 ||
		len(p.els(SelectorAnthologyItem)) == 0 {
		t.Skip("need both stories and anthologies to test type filter")
	}

	for _, filter := range []string{"Stories", "Anthologies", "All"} {
		btn := findFilterSegment(p, filter)
		require.NotNil(t, btn, "expected %s filter segment", filter)
		btn.MustClick()

		// Wait for HTMX swap to complete by checking for active state
		var ariaPressed *string
		require.Eventually(t, func() bool {
			btn = findFilterSegment(p, filter)
			if btn == nil {
				return false
			}
			ariaPressed = btn.MustAttribute("aria-pressed")
			return ariaPressed != nil && *ariaPressed == "true"
		}, defaultTimeout, 100*time.Millisecond, "%s should become active after click", filter)
	}
}

// testCategoryBooleanFilters tests the boolean filters (Unread, Starred).
func testCategoryBooleanFilters(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)
	navigateToCategory(p)

	// Test Unread and Starred filter toggles
	for _, filter := range []string{"Unread", "Starred"} {
		btn := findFilterButton(p, filter)
		require.NotNil(t, btn)

		ariaPressed := btn.MustAttribute("aria-pressed")
		assert.Equal(t, "false", *ariaPressed, "%s should start inactive", filter)

		btn.MustClick()
		p.waitStable()

		btn = findFilterButton(p, filter)
		ariaPressed = btn.MustAttribute("aria-pressed")
		assert.Equal(t, "true", *ariaPressed, "%s should be active after click", filter)
	}

	// Verify both filters remain active
	for _, filter := range []string{"Unread", "Starred"} {
		btn := findFilterButton(p, filter)
		ariaPressed := btn.MustAttribute("aria-pressed")
		assert.Equal(t, "true", *ariaPressed, "%s should remain active", filter)
	}
}

// testCategoryHiddenFilter tests the hidden filter on the category page.
func testCategoryHiddenFilter(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)
	navigateToCategory(p)

	ensureHiddenItem(p)

	hiddenItems := p.els(SelectorListItem + "[data-hidden]")
	require.NotEmpty(t, hiddenItems, "should have hidden items")

	// Verify hidden items not visible
	for _, item := range hiddenItems {
		box := item.MustShape().Box()
		assert.LessOrEqual(t, box.Height, 1.0, "hidden item should have minimal height when filter is off")
	}

	// Enable hidden filter
	findFilterButton(p, "Hidden").MustClick()
	p.waitStable()

	// Verify hidden items now visible
	for _, item := range p.els(SelectorListItem + "[data-hidden]") {
		box := item.MustShape().Box()
		assert.Greater(t, box.Height, 1.0, "hidden item should be visible when filter is on")
	}
}

// resourceTestCase defines a test case for resource-specific tests.
type resourceTestCase struct {
	name           string
	navigate       func(p *testPage) bool // returns false if skip
	itemSelector   string
	backSelector   string // selector for navigating back (empty = first breadcrumb)
	backLinkIndex  int    // which breadcrumb link to click (0 = first)
	contentElement string // element to verify on content page
}

var resourceTestCases = []resourceTestCase{
	{
		name: "Story",
		navigate: func(p *testPage) bool {
			navigateToCategory(p)
			return true
		},
		itemSelector:   VisibleItemByKind(component.KindStory),
		backSelector:   SelectorBreadcrumbs + " a",
		backLinkIndex:  0,
		contentElement: "article",
	},
	{
		name:           "Chapter",
		navigate:       navigateToAnthology,
		itemSelector:   SelectorChapterItem,
		backSelector:   SelectorBreadcrumbs + " a",
		backLinkIndex:  1, // Second breadcrumb link is the anthology
		contentElement: "article",
	},
	{
		name: "Anthology",
		navigate: func(p *testPage) bool {
			navigateToCategory(p)
			return true
		},
		itemSelector:   VisibleItemByKind(component.KindAnthology),
		backSelector:   SelectorBreadcrumbs + " a",
		backLinkIndex:  0,
		contentElement: "div[role='list']",
	},
}

// testAutoView tests that visiting a resource marks it as viewed.
func testAutoView(t *testing.T, newPage func(*testing.T) *testPage) {
	for _, tc := range resourceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPage(t)

			if !tc.navigate(p) {
				t.Skipf("no %ss found", strings.ToLower(tc.name))
			}

			items := p.els(tc.itemSelector)
			if len(items) == 0 {
				t.Skipf("no %ss found", strings.ToLower(tc.name))
			}

			// Get the item's ID and click its link
			item := items[0]
			itemID, err := item.Attribute("id")
			require.NoError(t, err)
			require.NotNil(t, itemID)

			item.Timeout(defaultTimeout).MustElement("a").MustClick()
			p.waitStable()

			// Verify we're on the content page
			p.el(tc.contentElement)

			// Navigate back (breadcrumbs are full page navigation, not HTMX)
			backLinks := p.els(tc.backSelector)
			require.Greater(t, len(backLinks), tc.backLinkIndex, "should have enough breadcrumb links")
			backLinks[tc.backLinkIndex].MustClick()
			p.waitLoad()

			// Check view toggle is now pressed - use attribute selector for IDs with slashes
			selector := fmt.Sprintf("article[id='%s']", *itemID)
			itemAfter := p.elMaybe(selector)
			require.NotNil(t, itemAfter, "should find %s after navigating back (selector: %s)", tc.name, selector)

			viewToggle, err := itemAfter.Timeout(defaultTimeout).Element(SelectorViewButton)
			require.NoError(t, err, "should find view toggle for %s", tc.name)

			ariaPressed := viewToggle.MustAttribute("aria-pressed")
			assert.Equal(t, "true", *ariaPressed, "%s should be marked as viewed after visiting", tc.name)
		})
	}
}

// testUpdatedTimestamp tests that resources show "updated" when update_time > view_time.
func testUpdatedTimestamp(t *testing.T, newPage func(*testing.T) *testPage, server *Server) {
	for _, tc := range resourceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPage(t)

			if !tc.navigate(p) {
				t.Skipf("no %ss found", strings.ToLower(tc.name))
			}

			items := p.els(tc.itemSelector)
			if len(items) == 0 {
				t.Skipf("no %ss found", strings.ToLower(tc.name))
			}

			// Get the item's ID
			item := items[0]
			itemID, err := item.Attribute("id")
			require.NoError(t, err)
			require.NotNil(t, itemID)

			// Set view_time to 2 years ago (before any update_time from dev service)
			oldViewTime := time.Now().AddDate(-2, 0, 0)
			err = server.SetViewTime(t.Context(), *itemID, oldViewTime)
			require.NoError(t, err)

			// Reload the page to see the updated timestamp
			p.reload()

			// Check timestamp label
			itemAfter := p.el(fmt.Sprintf("article[id='%s']", *itemID))
			timestamp := itemAfter.Timeout(defaultTimeout).MustElement("time")
			dataLabel := timestamp.MustAttribute("data-label")
			assert.Equal(t, "updated", *dataLabel, "%s with old view_time should show 'updated'", tc.name)
		})
	}
}

// testStarToggle tests that the star button toggles the starred state.
func testStarToggle(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)
	navigateToCategory(p)

	// Find an entry to test with and get its ID
	items := p.els(SelectorListItem + ":not([data-hidden])")
	require.NotEmpty(t, items, "expected at least one entry")
	itemID, err := items[0].Attribute("id")
	require.NoError(t, err)
	require.NotNil(t, itemID)

	itemSelector := fmt.Sprintf("article[id='%s']", *itemID)

	// Get initial star state
	item := p.el(itemSelector)
	starBtn := item.Timeout(defaultTimeout).MustElement(SelectorStarButton)
	initialPressed := starBtn.MustAttribute("aria-pressed")
	require.NotNil(t, initialPressed)

	// Click to toggle
	starBtn.MustClick()
	p.waitStable()

	// Re-query both item and button after HTMX update
	item = p.el(itemSelector)
	starBtn = item.Timeout(defaultTimeout).MustElement(SelectorStarButton)
	afterPressed := starBtn.MustAttribute("aria-pressed")
	require.NotNil(t, afterPressed)

	if *initialPressed == "false" {
		assert.Equal(t, "true", *afterPressed, "star should be pressed after clicking unstarred item")
	} else {
		assert.Equal(t, "false", *afterPressed, "star should be unpressed after clicking starred item")
	}

	// Toggle back and verify
	starBtn.MustClick()
	p.waitStable()

	// Re-query again
	item = p.el(itemSelector)
	starBtn = item.Timeout(defaultTimeout).MustElement(SelectorStarButton)
	finalPressed := starBtn.MustAttribute("aria-pressed")
	assert.Equal(t, *initialPressed, *finalPressed, "star state should return to initial after second toggle")
}

// testChapterPage tests navigating to a chapter and viewing content.
func testChapterPage(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)

	if !navigateToChapter(p) {
		t.Skip("no chapters found")
	}

	// Verify content loads
	content := p.el("article")
	require.NotNil(t, content)
	assert.NotEmpty(t, content.MustText(), "expected chapter content to have text")

	// Verify breadcrumbs show anthology
	// Should have at least 3 links: Archive > Category > Anthology
	breadcrumbLinks := p.els(SelectorBreadcrumbs + " a")
	assert.GreaterOrEqual(t, len(breadcrumbLinks), 3, "chapter should have at least 3 breadcrumb links")
}

// testFilterCombination tests that multiple filters work together correctly.
func testFilterCombination(t *testing.T, newPage func(*testing.T) *testPage) {
	p := newPage(t)
	navigateToCategory(p)

	// Ensure we have both stories and anthologies, and at least one hidden item
	stories := p.els(SelectorStoryItem)
	anthologies := p.els(SelectorAnthologyItem)
	if len(stories) == 0 || len(anthologies) == 0 {
		t.Skip("need both stories and anthologies to test filter combination")
	}

	// Hide a story if none are hidden
	hiddenStories := p.els(HiddenItemByKind(component.KindStory))
	if len(hiddenStories) == 0 {
		stories[0].Timeout(defaultTimeout).MustElement(SelectorHideButton).MustClick()
		p.waitStable()
	}

	// Apply Stories filter
	storiesBtn := findFilterSegment(p, "Stories")
	require.NotNil(t, storiesBtn)
	storiesBtn.MustClick()
	p.waitStable()

	// Should see only stories (minus hidden ones)
	visibleItems := p.els(SelectorListItem + ":not([data-hidden])")
	for _, item := range visibleItems {
		kind, err := item.Attribute(component.DataAttrKind)
		require.NoError(t, err)
		// Hidden items have minimal height, visible ones are stories
		box := item.MustShape().Box()
		if box.Height > 1.0 {
			assert.Equal(t, component.KindStory, *kind, "visible items should be stories when Stories filter is active")
		}
	}

	// Enable Hidden filter as well
	findFilterButton(p, "Hidden").MustClick()
	p.waitStable()

	// Now hidden stories should be visible too
	hiddenStories = p.els(HiddenItemByKind(component.KindStory))
	for _, item := range hiddenStories {
		box := item.MustShape().Box()
		assert.Greater(t, box.Height, 1.0, "hidden stories should be visible when both Stories and Hidden filters are active")
	}
}

// testPaginationNavigation tests that pagination works correctly.
func testPaginationNavigation(t *testing.T, newPage func(*testing.T) *testPage, _ *Server) {
	p := newPage(t)
	navigateToCategory(p)

	// Get the first entry on page 1
	firstEntryPage1 := p.el(SelectorListItem + ":first-child")
	firstEntryID := firstEntryPage1.MustAttribute("id")
	require.NotNil(t, firstEntryID, "first entry should have an ID")

	// Find and click the Next button
	nextBtn := p.elMaybe(SelectorPagination + " a")
	if nextBtn == nil {
		t.Skip("pagination not available (fewer than 100 entries)")
	}

	// Verify URL doesn't have page param yet
	url1 := p.MustInfo().URL
	assert.NotContains(t, url1, "page=", "page 1 should not have page param in URL")

	nextBtn.MustClick()
	p.waitLoad()

	// Verify URL now has page param
	url2 := p.MustInfo().URL
	assert.Contains(t, url2, "page=", "page 2 should have page param in URL")

	// Verify different content is shown
	firstEntryPage2 := p.el(SelectorListItem + ":first-child")
	firstEntryID2 := firstEntryPage2.MustAttribute("id")
	require.NotNil(t, firstEntryID2, "first entry on page 2 should have an ID")
	assert.NotEqual(t, *firstEntryID, *firstEntryID2, "page 2 should show different entries than page 1")

	// Use browser back and verify we return to page 1
	p.MustNavigateBack()
	p.waitLoad()

	firstEntryAfterBack := p.el(SelectorListItem + ":first-child")
	firstEntryIDAfterBack := firstEntryAfterBack.MustAttribute("id")
	assert.Equal(t, *firstEntryID, *firstEntryIDAfterBack, "browser back should return to page 1")
}

// testBreadcrumbFilterPropagation tests that breadcrumbs preserve filter state from parent pages.
func testBreadcrumbFilterPropagation(t *testing.T, newPage func(*testing.T) *testPage, _ *Server) {
	p := newPage(t)
	navigateToCategory(p)

	// Apply a filter (Stories)
	storiesBtn := findFilterSegment(p, "Stories")
	if storiesBtn == nil {
		t.Skip("Stories filter segment not found (may not have both entry types)")
	}
	storiesBtn.MustClick()
	p.waitStable()

	// Verify URL has filter param
	categoryURL := p.MustInfo().URL
	assert.Contains(t, categoryURL, "type=story", "category URL should have type filter")

	// Click on a story to navigate to it
	storyLink := p.elMaybe(VisibleItemByKind(component.KindStory) + " > a")
	if storyLink == nil {
		t.Skip("no visible stories found")
	}
	storyLink.MustClick()
	p.waitLoad()

	// Verify we're on the story page (URL has pf param with parent filter state)
	storyURL := p.MustInfo().URL
	assert.Contains(t, storyURL, "pf=", "story URL should have parent filter state")

	// Find the category breadcrumb and verify it has the filter params
	categoryBreadcrumb := p.el(SelectorBreadcrumbs + " a:first-of-type")
	require.NotNil(t, categoryBreadcrumb)
	breadcrumbHref := categoryBreadcrumb.MustAttribute("href")
	require.NotNil(t, breadcrumbHref)
	assert.Contains(t, *breadcrumbHref, "type=story", "category breadcrumb should preserve filter state")

	// Click the breadcrumb and verify we return to filtered category
	categoryBreadcrumb.MustClick()
	p.waitLoad()

	// Verify we're back on the category with the filter still applied
	returnURL := p.MustInfo().URL
	assert.Contains(t, returnURL, "type=story", "returning via breadcrumb should preserve filter")

	// Verify the Stories filter is still active
	storiesBtn = findFilterSegment(p, "Stories")
	require.NotNil(t, storiesBtn)
	ariaPressed := storiesBtn.MustAttribute("aria-pressed")
	assert.Equal(t, "true", *ariaPressed, "Stories filter should still be active after breadcrumb navigation")
}

// testMarkReadFilterPreservation tests that "Mark Read & Return" preserves filter state.
func testMarkReadFilterPreservation(t *testing.T, newPage func(*testing.T) *testPage, server *Server) {
	p := newPage(t)
	navigateToCategory(p)

	// Apply Stories filter
	storiesBtn := findFilterSegment(p, "Stories")
	require.NotNil(t, storiesBtn, "Stories filter segment should exist")
	storiesBtn.MustClick()
	p.waitStable()

	// Verify filter is applied in URL
	categoryURL := p.MustInfo().URL
	assert.Contains(t, categoryURL, "type=story", "should have type filter in URL")

	// Click on the first visible story
	storyLink := p.el(VisibleItemByKind(component.KindStory) + " > a")
	storyLink.MustClick()
	p.waitLoad()

	// Find the footer return link (Mark Read & Return or Mark Unread & Return)
	footerLink := p.el("footer a")
	markReadHref := footerLink.MustAttribute("href")
	require.NotNil(t, markReadHref, "footer link should have href")

	// Verify the return link contains the parent's filter state
	assert.Contains(t, *markReadHref, "type=story", "return link should preserve type filter")

	// Navigate to the return URL and verify filters are preserved
	p.Page.Timeout(defaultTimeout).MustNavigate(server.URL(*markReadHref)).MustWaitLoad()

	returnURL := p.MustInfo().URL
	assert.Contains(t, returnURL, "type=story", "should return with type filter in URL")

	// Verify filter is still visually active
	storiesBtn = findFilterSegment(p, "Stories")
	require.NotNil(t, storiesBtn)
	assert.Equal(t, "true", *storiesBtn.MustAttribute("aria-pressed"), "Stories filter should still be active")
}

// contentToggleTestCase defines a test case for content page toggle tests.
type contentToggleTestCase struct {
	name         string
	navigate     func(p *testPage) bool
	toggleAction component.Action
}

var contentToggleTestCases = []contentToggleTestCase{
	{
		name:         "Story",
		navigate:     navigateToStory,
		toggleAction: component.ActionStar,
	},
	{
		name:         "Chapter",
		navigate:     navigateToChapter,
		toggleAction: component.ActionView,
	},
}

// testContentPageToggle tests that toggle buttons on content pages (story/chapter)
// correctly update only the action buttons, not the site header.
// Regression test for HTMX misconfiguration where clicking toggles replaced action buttons with header content.
func testContentPageToggle(t *testing.T, newPage func(*testing.T) *testPage) {
	for _, testCase := range contentToggleTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := newPage(t)

			if !testCase.navigate(p) {
				t.Skipf("no %ss found", strings.ToLower(testCase.name))
			}

			// Verify initial state: site header exists with site title
			require.NotNil(t, p.el(SelectorSiteHeaderNav))
			siteTitleBefore := p.el(SelectorSiteHeader + " " + SelectorSiteTitle).MustText()

			// Find the content header's action nav and count buttons
			require.NotNil(t, p.el(SelectorContentActions), "content header should have action nav")
			buttonCountBefore := len(p.els(SelectorContentActions + " > button"))
			require.Positive(t, buttonCountBefore, "should have action buttons")

			// Get the toggle button's initial state
			toggleSelector := ContentActionButton(string(testCase.toggleAction))
			toggleBtn := p.el(toggleSelector)
			initialState := toggleBtn.MustAttribute("aria-pressed")
			require.NotNil(t, initialState)

			// Click the toggle button and wait for HTMX request to complete
			p.clickAndWaitIdle(toggleBtn)

			// Verify site header is unchanged (re-query after HTMX swap)
			siteTitleAfter := p.el(SelectorSiteHeader + " " + SelectorSiteTitle).MustText()
			assert.Equal(t, siteTitleBefore, siteTitleAfter, "site header should be unchanged after toggle")

			// Verify action nav still exists with same number of buttons
			require.NotNil(t, p.elMaybe(SelectorContentActions),
				"content header should still have action nav after toggle")
			assert.Len(t, p.els(SelectorContentActions+" > button"), buttonCountBefore,
				"should have same number of action buttons after toggle")

			// Verify the toggle state changed (re-query after HTMX swap)
			toggleBtnAfter := p.el(toggleSelector)
			finalState := toggleBtnAfter.MustAttribute("aria-pressed")
			assert.NotEqual(t, *initialState, *finalState, "toggle state should have changed after click")
		})
	}
}
