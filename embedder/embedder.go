package embedder

// Embedder embeds a batch of texts in one call, returning one embedding per
// input text in the same order.
type Embedder interface {
	Embed(texts []string) ([][]float32, error)
}
