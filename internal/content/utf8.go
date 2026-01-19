package content

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/saintfish/chardet"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
)

// utf8BOM is the UTF-8 byte order mark that some editors add to files.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// minChardetConfidence is the minimum confidence level required to trust
// chardet's detection over the default Windows-1252 fallback.
const minChardetConfidence = 50

// UTF8Transformer converts input to UTF-8 based on the content type charset.
// It also strips the UTF-8 BOM if present.
//
// Detection strategy:
//  1. Use charset.DetermineEncoding (checks BOM, Content-Type, meta tags)
//  2. If detection is uncertain and content is plain text, use chardet for
//     statistical detection of non-UTF-8 encodings
//  3. Convert to UTF-8 and strip BOM
func UTF8Transformer(contentType string) TransformerFunc {
	isPlainText := strings.HasPrefix(contentType, "text/plain")

	return func(input []byte) ([]byte, error) {
		enc, name, certain := charset.DetermineEncoding(input, contentType)

		// When detection is uncertain for plain text, try statistical detection
		if !certain && isPlainText {
			if detectedEnc, detectedName := detectWithChardet(input); detectedEnc != nil {
				enc, name = detectedEnc, detectedName
			}
		}

		if !certain {
			slog.Debug("encoding detection uncertain",
				slog.String("encoding", name),
				slog.String("content_type", contentType))
		}

		output, err := decodeToUTF8(input, enc)
		if err != nil {
			return nil, err
		}
		return bytes.TrimPrefix(output, utf8BOM), nil
	}
}

// detectWithChardet uses ICU-based statistical detection for plain text.
// Returns nil if detection fails or confidence is too low.
func detectWithChardet(input []byte) (encoding.Encoding, string) {
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(input)
	if err != nil || result.Confidence < minChardetConfidence {
		return nil, ""
	}

	// Map chardet's charset name to an encoding
	enc, err := htmlindex.Get(result.Charset)
	if err != nil {
		// chardet sometimes returns names not in the HTML index
		// Fall back to the original detection
		return nil, ""
	}

	slog.Debug("chardet detection",
		slog.String("charset", result.Charset),
		slog.Int("confidence", result.Confidence))

	return enc, result.Charset
}

// decodeToUTF8 converts input bytes to UTF-8 using the given encoding.
func decodeToUTF8(input []byte, enc encoding.Encoding) ([]byte, error) {
	// UTF-8 and Nop encodings don't need conversion
	if enc == encoding.Nop || enc == unicode.UTF8 {
		return input, nil
	}

	reader := enc.NewDecoder().Reader(bytes.NewReader(input))
	output, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode to UTF-8: %w", err)
	}
	return output, nil
}
