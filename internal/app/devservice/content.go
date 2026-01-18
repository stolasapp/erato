package devservice

import (
	"fmt"
	"strings"

	"github.com/brianvoe/gofakeit/v7"
)

// Content generation constants.
const (
	minParagraphs        = 3
	maxExtraPara         = 8 // 3-10 paragraphs total
	minSentences         = 3
	maxExtraSent         = 5 // 3-7 sentences total
	minWords             = 8
	maxExtraWords        = 12   // 8-20 words total
	plainTextProbability = 0.15 // 15% plain text, 85% HTML
)

// generateContent creates random content that is either HTML (85%) or plain text (15%).
// Returns the content and whether it is plain text.
func generateContent(faker *gofakeit.Faker) (content string, isPlainText bool) {
	numParagraphs := minParagraphs + faker.IntN(maxExtraPara)
	paragraphs := make([]string, numParagraphs)

	for i := range numParagraphs {
		paragraphs[i] = generateParagraph(faker)
	}

	isPlainText = faker.Float64() < plainTextProbability
	if isPlainText {
		return strings.Join(paragraphs, "\n\n"), true
	}

	var builder strings.Builder
	for _, p := range paragraphs {
		builder.WriteString("<p>")
		builder.WriteString(p)
		builder.WriteString("</p>\n")
	}
	return builder.String(), false
}

func generateParagraph(faker *gofakeit.Faker) string {
	numSentences := minSentences + faker.IntN(maxExtraSent)
	sentences := make([]string, numSentences)
	for i := range numSentences {
		sentences[i] = faker.Sentence(minWords + faker.IntN(maxExtraWords))
	}
	return strings.Join(sentences, " ")
}

// Word lists for generating titles and chapter names.
var categoryNames = []string{
	"Fantasy", "Science Fiction", "Mystery", "Romance", "Horror",
	"Adventure", "Drama", "Comedy", "Thriller", "Historical",
	"Supernatural", "Contemporary", "Action", "Slice of Life",
}

var categoryDescriptions = []string{
	"A collection of tales from distant realms.",
	"Stories that push the boundaries of imagination.",
	"Narratives exploring the depths of human experience.",
	"Adventures that span across time and space.",
	"Tales of mystery, intrigue, and wonder.",
}

var chapterTitles = []string{
	"The Beginning", "A New Dawn", "Into the Unknown",
	"The Meeting", "Revelations", "The Journey Continues",
	"Dark Times", "A Ray of Hope", "The Confrontation",
	"Secrets Unveiled", "The Path Forward", "Unexpected Allies",
	"The Gathering Storm", "Calm Before", "The Final Stand",
	"Aftermath", "New Horizons", "The Return", "Closure", "Epilogue",
}

func generateTitle(faker *gofakeit.Faker) string {
	patterns := []func(*gofakeit.Faker) string{
		func(f *gofakeit.Faker) string { return fmt.Sprintf("The %s %s", f.Adjective(), f.Noun()) },
		func(f *gofakeit.Faker) string { return fmt.Sprintf("A %s of %s", f.Noun(), f.Noun()) },
		func(f *gofakeit.Faker) string {
			return fmt.Sprintf("%s and %s", titleCase(f.Noun()), titleCase(f.Noun()))
		},
		func(f *gofakeit.Faker) string { return fmt.Sprintf("The %s's %s", f.Noun(), f.Noun()) },
		func(f *gofakeit.Faker) string {
			return fmt.Sprintf("%s of the %s", titleCase(f.Noun()), f.Adjective())
		},
	}
	return patterns[faker.IntN(len(patterns))](faker)
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
