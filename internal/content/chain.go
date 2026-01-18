package content

// Chain chains together a set of transformers, failing fast if any transformer
// in the chain errors.
func Chain(transformers ...Transformer) TransformerFunc {
	return func(input []byte) ([]byte, error) {
		var err error
		for _, transformer := range transformers {
			input, err = transformer.Transform(input)
			if err != nil {
				return nil, err
			}
		}
		return input, nil
	}
}
