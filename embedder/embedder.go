package embedder

type Embedder interface {
	Embed(text string) ([]float32, error)
}
