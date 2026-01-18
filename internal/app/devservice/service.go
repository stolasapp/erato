// Package devservice provides a fake upstream HTTP server for development and testing.
// It serves HTML pages that match the structure expected by the archive scraper.
package devservice

import (
	"cmp"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// Corpus generation constants.
const (
	minCategories        = 8
	maxExtraCategories   = 5 // 8-12 categories total
	minEntries           = 120
	maxExtraEntries      = 40 // 120-160 entries total (ensures pagination with 100/page)
	minChapters          = 1
	maxExtraChapters     = 20 // 1-20 chapters total
	anthologyProbability = 0.3
)

// Seed returns the dev service seed from the DEV_SERVICE_SEED environment
// variable, or a random value if not set.
func Seed() uint64 {
	if env := os.Getenv("DEV_SERVICE_SEED"); env != "" {
		if seed, err := strconv.ParseUint(env, 10, 64); err == nil {
			return seed
		}
	}
	return rand.Uint64() //nolint:gosec // intentionally weak random for test data
}

// category represents a generated category.
type category struct {
	slug        string
	displayName string
	description string
}

// chapter represents a generated chapter.
type chapter struct {
	slug        string
	updateTime  time.Time
	content     string
	isPlainText bool
}

// entry represents a generated entry.
type entry struct {
	slug        string
	displayName string
	isAnthology bool
	updateTime  time.Time
	content     string    // for stories
	isPlainText bool      // for stories
	chapters    []chapter // for anthologies
}

// Service is an HTTP server that serves fake archive pages for the scraper.
type Service struct {
	mux        *http.ServeMux
	categories []category
	entries    map[string][]entry // keyed by category slug
}

// New creates a new dev service with a seeded random corpus.
func New(seed uint64) *Service {
	faker := gofakeit.New(seed)
	svc := &Service{
		mux:     http.NewServeMux(),
		entries: make(map[string][]entry),
	}
	svc.generateCorpus(faker)
	svc.registerRoutes()
	return svc
}

// ServeHTTP satisfies [http.Handler].
func (s *Service) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mux.ServeHTTP(writer, request)
}

func (s *Service) generateCorpus(faker *gofakeit.Faker) {
	numCategories := minCategories + faker.IntN(maxExtraCategories)

	for i := range numCategories {
		cat := s.generateCategory(faker, i)
		s.categories = append(s.categories, cat)

		numEntries := minEntries + faker.IntN(maxExtraEntries)
		for range numEntries {
			ent := s.generateEntry(faker)
			s.entries[cat.slug] = append(s.entries[cat.slug], ent)
		}

		// Sort entries by updateTime descending
		slices.SortFunc(s.entries[cat.slug], func(a, b entry) int {
			return b.updateTime.Compare(a.updateTime)
		})
	}

	// Sort categories alphabetically by displayName
	slices.SortFunc(s.categories, func(a, b category) int {
		return cmp.Compare(a.displayName, b.displayName)
	})
}

func (s *Service) generateCategory(faker *gofakeit.Faker, index int) category {
	name := categoryNames[index%len(categoryNames)]
	if index >= len(categoryNames) {
		name = fmt.Sprintf("%s %d", name, index/len(categoryNames)+1)
	}
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	return category{
		slug:        slug,
		displayName: name,
		description: categoryDescriptions[faker.IntN(len(categoryDescriptions))],
	}
}

func (s *Service) generateEntry(faker *gofakeit.Faker) entry {
	title := generateTitle(faker)
	slug := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	slug = strings.ReplaceAll(slug, "'", "")

	isAnthology := faker.Float64() < anthologyProbability
	updateTime := faker.DateRange(
		time.Now().AddDate(-1, 0, 0),
		time.Now(),
	)

	ent := entry{
		slug:        slug,
		displayName: title,
		isAnthology: isAnthology,
		updateTime:  updateTime,
	}

	if isAnthology {
		numChapters := minChapters + faker.IntN(maxExtraChapters)
		ent.chapters = make([]chapter, 0, numChapters)
		for i := range numChapters {
			chapterTitle := chapterTitles[faker.IntN(len(chapterTitles))]
			chapterSlug := fmt.Sprintf("chapter-%d-%s", i+1, strings.ToLower(strings.ReplaceAll(chapterTitle, " ", "-")))
			chapterTime := faker.DateRange(
				time.Now().AddDate(-1, 0, 0),
				time.Now(),
			)
			chapterContent, chapterIsPlainText := generateContent(faker)
			ent.chapters = append(ent.chapters, chapter{
				slug:        chapterSlug,
				updateTime:  chapterTime,
				content:     chapterContent,
				isPlainText: chapterIsPlainText,
			})
		}
		// Sort chapters by updateTime descending
		slices.SortFunc(ent.chapters, func(a, b chapter) int {
			return b.updateTime.Compare(a.updateTime)
		})
	} else {
		ent.content, ent.isPlainText = generateContent(faker)
	}

	return ent
}

func (s *Service) registerRoutes() {
	// Root page - list of categories
	s.mux.HandleFunc("GET /{$}", s.handleRoot)

	// Category page - list of entries
	s.mux.HandleFunc("GET /{category}/", s.handleCategory)

	// Entry page - content (story) or list of chapters (anthology)
	s.mux.HandleFunc("GET /{category}/{entry}/", s.handleEntry)

	// Chapter page - content
	s.mux.HandleFunc("GET /{category}/{entry}/{chapter}/", s.handleChapter)
}

func (s *Service) handleRoot(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))

	writeString(writer, `<!DOCTYPE html><html><head><title>Archive</title></head><body>`)
	for _, cat := range s.categories {
		writef(writer, `<div class="list-group-item"><a href="/%s/">%s</a> - %s</div>`,
			cat.slug, cat.displayName, cat.description)
	}
	writeString(writer, `</body></html>`)
}

func (s *Service) handleCategory(writer http.ResponseWriter, request *http.Request) {
	categorySlug := request.PathValue("category")

	entries, ok := s.entries[categorySlug]
	if !ok {
		http.NotFound(writer, request)
		return
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))

	writeString(writer, `<!DOCTYPE html><html><head><title>Category</title></head><body><table>`)
	writeString(writer, `<tr><th>Type</th><th>Date</th><th>Name</th></tr>`)
	for _, ent := range entries {
		entryType := "File"
		if ent.isAnthology {
			entryType = "Dir"
		}
		writef(writer, `<tr><td>%s</td><td>%s</td><td><a href="%s/">%s</a></td></tr>`,
			entryType,
			formatTimestamp(ent.updateTime),
			ent.slug,
			ent.displayName)
	}
	writeString(writer, `</table></body></html>`)
}

func (s *Service) handleEntry(writer http.ResponseWriter, request *http.Request) {
	categorySlug := request.PathValue("category")
	entrySlug := request.PathValue("entry")

	entries, ok := s.entries[categorySlug]
	if !ok {
		http.NotFound(writer, request)
		return
	}

	var ent *entry
	for i := range entries {
		if entries[i].slug == entrySlug {
			ent = &entries[i]
			break
		}
	}
	if ent == nil {
		http.NotFound(writer, request)
		return
	}

	writer.Header().Set("Last-Modified", ent.updateTime.Format(time.RFC1123))

	if ent.isAnthology {
		// Anthology - show chapter list (already sorted by updateTime descending)
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writeString(writer, `<!DOCTYPE html><html><head><title>Anthology</title></head><body><table>`)
		writeString(writer, `<tr><th>Type</th><th>Date</th><th>Name</th></tr>`)
		for _, ch := range ent.chapters {
			writef(writer, `<tr><td>File</td><td>%s</td><td><a href="%s/">%s</a></td></tr>`,
				formatTimestamp(ch.updateTime),
				ch.slug,
				ch.slug)
		}
		writeString(writer, `</table></body></html>`)
	} else {
		// Story - show content
		if ent.isPlainText {
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			writeString(writer, ent.content)
		} else {
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			writef(writer, `<!DOCTYPE html><html><head><title>%s</title></head><body>%s</body></html>`,
				ent.displayName, ent.content)
		}
	}
}

func (s *Service) handleChapter(writer http.ResponseWriter, request *http.Request) {
	categorySlug := request.PathValue("category")
	entrySlug := request.PathValue("entry")
	chapterSlug := request.PathValue("chapter")

	entries, ok := s.entries[categorySlug]
	if !ok {
		http.NotFound(writer, request)
		return
	}

	var ent *entry
	for i := range entries {
		if entries[i].slug == entrySlug {
			ent = &entries[i]
			break
		}
	}
	if ent == nil || !ent.isAnthology {
		http.NotFound(writer, request)
		return
	}

	var chap *chapter
	for i := range ent.chapters {
		if ent.chapters[i].slug == chapterSlug {
			chap = &ent.chapters[i]
			break
		}
	}
	if chap == nil {
		http.NotFound(writer, request)
		return
	}

	writer.Header().Set("Last-Modified", chap.updateTime.Format(time.RFC1123))

	if chap.isPlainText {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writeString(writer, chap.content)
	} else {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writef(writer, `<!DOCTYPE html><html><head><title>%s</title></head><body>%s</body></html>`,
			chap.slug, chap.content)
	}
}

func formatTimestamp(timestamp time.Time) string {
	now := time.Now()
	if timestamp.Year() == now.Year() {
		// Recent format: "Jan _2 15:04"
		return timestamp.Format("Jan _2 15:04")
	}
	// Older format: "Jan _2 2006"
	return timestamp.Format("Jan _2 2006")
}

// writeString writes a string to the writer, discarding any error.
// Errors are ignored since this is test/dev infrastructure where write failures
// are unrecoverable and will manifest as test failures anyway.
func writeString(writer io.Writer, str string) {
	_, _ = io.WriteString(writer, str)
}

// writef writes a formatted string to the writer, discarding any error.
// See writeString for rationale on discarded errors.
func writef(writer io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(writer, format, args...)
}
