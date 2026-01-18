package content

// Transformer modifies content, returning modified content or an error.
type Transformer interface {
	// Transform modifies input, returning modified content or an error.
	Transform(input []byte) ([]byte, error)
}

// TransformerFunc is a [Transformer] that can be represented just by the
// [Transform] method.
type TransformerFunc func(input []byte) ([]byte, error)

// Transform satisfies [Transformer].
func (fn TransformerFunc) Transform(input []byte) ([]byte, error) { return fn(input) }
