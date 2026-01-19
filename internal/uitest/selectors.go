package uitest

import (
	"fmt"

	"github.com/stolasapp/erato/internal/app/component"
)

// CSS selectors built from component constants.
// These ensure test selectors stay in sync with the component DOM structure.

// Element selectors.
var (
	// SelectorListContainer selects the list container by ID.
	SelectorListContainer = "#" + component.IDListContainer

	// SelectorContentActions selects the content page action nav by ID.
	SelectorContentActions = "#" + component.IDContentActions

	// SelectorSiteHeader selects the site header by class.
	SelectorSiteHeader = "header." + component.ClassSiteHeader

	// SelectorSiteHeaderNav selects the nav inside the site header.
	SelectorSiteHeaderNav = SelectorSiteHeader + " nav"

	// SelectorSiteTitle selects the site title by class.
	SelectorSiteTitle = "." + component.ClassSiteTitle

	// SelectorFilters selects the filter bar by class.
	SelectorFilters = "nav." + component.ClassFilters

	// SelectorPagination selects the pagination nav by class.
	SelectorPagination = "nav." + component.ClassPagination

	// SelectorBreadcrumbs selects the breadcrumbs container by class.
	SelectorBreadcrumbs = "." + component.ClassBreadcrumbs
)

// List item selectors by kind.
var (
	// SelectorListItem selects any list item article.
	SelectorListItem = "div[role='list'] > article"

	// SelectorStoryItem selects story list items.
	SelectorStoryItem = selectorByKind(component.KindStory)

	// SelectorAnthologyItem selects anthology list items.
	SelectorAnthologyItem = selectorByKind(component.KindAnthology)

	// SelectorChapterItem selects chapter list items.
	SelectorChapterItem = selectorByKind(component.KindChapter)

	// SelectorCategoryItem selects category list items.
	SelectorCategoryItem = selectorByKind(component.KindCategory)
)

// Action button selectors.
var (
	// SelectorViewButton selects the view toggle button.
	SelectorViewButton = selectorByAction(string(component.ActionView))

	// SelectorReadButton selects the read toggle button.
	SelectorReadButton = selectorByAction(string(component.ActionRead))

	// SelectorStarButton selects the star toggle button.
	SelectorStarButton = selectorByAction(string(component.ActionStar))

	// SelectorHideButton selects the hide toggle button.
	SelectorHideButton = selectorByAction(string(component.ActionHide))
)

// selectorByKind returns a selector for list items with a specific data-kind.
func selectorByKind(kind string) string {
	return fmt.Sprintf("div[role='list'] > article[%s='%s']", component.DataAttrKind, kind)
}

// selectorByAction returns a selector for buttons with a specific data-action.
func selectorByAction(action string) string {
	return fmt.Sprintf("button[%s='%s']", component.DataAttrAction, action)
}

// VisibleItemByKind returns a selector for visible (non-hidden) list items of a kind.
func VisibleItemByKind(kind string) string {
	return fmt.Sprintf("div[role='list'] > article[%s='%s']:not([%s])",
		component.DataAttrKind, kind, component.DataAttrHidden)
}

// HiddenItemByKind returns a selector for hidden list items of a kind.
func HiddenItemByKind(kind string) string {
	return fmt.Sprintf("div[role='list'] > article[%s='%s'][%s]",
		component.DataAttrKind, kind, component.DataAttrHidden)
}

// ContentActionButton returns a selector for an action button in the content header.
func ContentActionButton(action string) string {
	return fmt.Sprintf("main > header > nav > button[%s='%s']", component.DataAttrAction, action)
}
