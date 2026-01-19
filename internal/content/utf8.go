package content

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
)

// utf8BOM is the UTF-8 byte order mark that some editors add to files.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// UTF8Transformer converts input to UTF-8 based on the content type charset.
// It also strips the UTF-8 BOM if present.
func UTF8Transformer(contentType string) TransformerFunc {
	return func(input []byte) ([]byte, error) {
		reader, err := charset.NewReader(bytes.NewReader(input), contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to create charset reader: %w", err)
		}
		output, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return bytes.TrimPrefix(output, utf8BOM), nil
	}
}
