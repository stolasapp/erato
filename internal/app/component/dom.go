package component

// HTMX target element IDs.
const (
	IDContentActions = "content-actions"
	IDListContainer  = "list-container"
)

// HTMX target selectors.
const (
	TargetContentActions = "#" + IDContentActions
	TargetListContainer  = "#" + IDListContainer
	TargetClosestArticle = "closest article"
)

// Data attribute names (without the "data-" prefix for use in templ attributes).
const (
	AttrAction = "action"
	AttrKind   = "kind"
)

// Data attribute names with prefix (for use in CSS selectors and tests).
const (
	DataAttrAction = "data-" + AttrAction
	DataAttrKind   = "data-" + AttrKind
	DataAttrHidden = "data-hidden"
	DataAttrRead   = "data-read"
)

// Kind values for the data-kind attribute.
const (
	KindStory     = "story"
	KindAnthology = "anthology"
	KindChapter   = "chapter"
	KindCategory  = "category"
)

// CSS class names.
const (
	ClassSiteHeader  = "site-header"
	ClassSiteTitle   = "site-title"
	ClassFilters     = "filters"
	ClassPagination  = "pagination"
	ClassBreadcrumbs = "breadcrumbs"
)
