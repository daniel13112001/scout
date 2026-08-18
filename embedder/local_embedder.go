package embedder

type LocalEmbedder struct {
}

type EmbeddingResult struct {
	Embedding []float32
	Err       error
}

func (*LocalEmbedder) Embed(text string) ([]float32, error) {
	return []float32{0.0}, nil
}

func (e *LocalEmbedder) BatchEmbed(texts []string) []EmbeddingResult {
	results := make([]EmbeddingResult, len(texts))

	for i, text := range texts {
		embedding, err := e.Embed(text)

		results[i] = EmbeddingResult{
			Embedding: embedding,
			Err:       err,
		}
	}

	return results
}
