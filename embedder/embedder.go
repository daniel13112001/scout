package embedder

// Embedder embeds a batch of texts in one call, returning one embedding per
// input text in the same order.
type Embedder interface {
	Embed(texts []string) ([][]float32, error)

	// ModelID identifies the exact model producing embeddings, so stored
	// vectors can be detected as stale if the model ever changes.
	ModelID() string
}
