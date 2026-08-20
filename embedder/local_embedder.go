package embedder

import (
	"errors"
	"fmt"

	"github.com/daulet/tokenizers"
	ort "github.com/yalue/onnxruntime_go"
)

// EmbedderConfig configures a LocalEmbedder: where its model and tokenizer
// live on disk, and how it should run.
type EmbedderConfig struct {
	ModelPath      string
	TokenizerPath  string
	OrtLibraryPath string
	BatchSize      int
}

type EmbeddingResult struct {
	Embedding []float32
	Err       error
}

type LocalEmbedder struct {
	tokenizer *tokenizers.Tokenizer
	batchSize int

	// TODO: hold the ORT session here once inference is implemented.
}

func NewLocalEmbedder(cfg EmbedderConfig) (*LocalEmbedder, error) {
	tokenizer, err := tokenizers.FromFile(cfg.TokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("loading tokenizer: %w", err)
	}

	ort.SetSharedLibraryPath(cfg.OrtLibraryPath)
	if err := ort.InitializeEnvironment(); err != nil {
		tokenizer.Close()
		return nil, fmt.Errorf("initializing onnxruntime environment: %w", err)
	}

	// TODO: load cfg.ModelPath into an ORT session.

	return &LocalEmbedder{
		tokenizer: tokenizer,
		batchSize: cfg.BatchSize,
	}, nil
}

// Close releases the tokenizer and onnxruntime resources held by e.
func (e *LocalEmbedder) Close() error {
	defer ort.DestroyEnvironment()
	return e.tokenizer.Close()
}

// Embed is intentionally not implemented yet.
func (e *LocalEmbedder) Embed(text string) ([]float32, error) {
	return nil, errors.New("LocalEmbedder.Embed: not implemented")
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
