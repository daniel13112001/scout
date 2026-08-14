package embedder

type LocalEmbedder struct{

}

func (*LocalEmbedder) Embed(text string) ([]float32, error){
	return []float32{0.0}, nil
}