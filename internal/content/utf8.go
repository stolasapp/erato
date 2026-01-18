package content

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
)

// UTF8Transformer converts input to UTF-8 based on the content type charset.
func UTF8Transformer(contentType string) TransformerFunc {
	return func(input []byte) ([]byte, error) {
		reader, err := charset.NewReader(bytes.NewReader(input), contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to create charset reader: %w", err)
		}
		return io.ReadAll(reader)
	}
}
