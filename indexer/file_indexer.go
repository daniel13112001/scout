package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/daniel13112001/scout/embedder"
)

const (
	chunkSize    = 500
	fileWorkers  = 4
	writerBuffer = 64
)

type FileIndexer struct {
	Db       *sql.DB
	Embedder embedder.Embedder
}

type ProcessFileResult struct {
	FailedChunkErrors []error
}

func NewFileIndexer(db *sql.DB, embedder embedder.Embedder) (*FileIndexer, error) {
	if db == nil {
		return nil, errors.New("db is nil. cannot initialize file indexer")
	}

	if embedder == nil {
		return nil, errors.New("embedder is nil. cannot initialize file indexer")
	}

	return &FileIndexer{
		Db:       db,
		Embedder: embedder,
	}, nil
}

// IndexDirectory recursively walks a directory and processes each file.
func (fi *FileIndexer) IndexDirectory(dir string, extensions string) error {
	files := make(chan string)
	errCh := make(chan error, 1)

	// All file workers emit through this single writer, which is the only
	// goroutine that ever touches the database.
	writer := newDBWriter(fi.Db, writerBuffer)
	writer.start()

	var wg sync.WaitGroup

	// Start a bounded number of workers so a directory with many files
	// does not create an unbounded number of goroutines.
	for i := 0; i < fileWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for path := range files {
				_, err := fi.processFile(path, writer)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
				}
			}
		}()
	}

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// TODO: filter by allowed extensions.

		files <- path
		return nil
	})

	close(files)
	wg.Wait()

	writerErr := writer.close()

	if walkErr != nil {
		return walkErr
	}

	select {
	case err := <-errCh:
		return err
	default:
		return writerErr
	}
}

// processFile reads and chunks a file, embeds each chunk, and emits the
// resulting embeddings to the given Emitter for the single db writer to
// persist.
func (fi *FileIndexer) processFile(path string, emitter Emitter) (ProcessFileResult, error) {
	chunks, err := ChunkFile(path, chunkSize)
	if err != nil {
		return ProcessFileResult{}, err
	}

	res := ProcessFileResult{}

	for _, chunk := range chunks {
		embedding, err := fi.Embedder.Embed(chunk.content)
		if err != nil {
			res.FailedChunkErrors = append(
				res.FailedChunkErrors,
				fmt.Errorf("chunk %d: %w", chunk.index, err),
			)
			continue
		}

		if err := emitter.Emit(EmbeddingRecord{
			FilePath:   path,
			ChunkIndex: chunk.index,
			Content:    chunk.content,
			Embedding:  embedding,
		}); err != nil {
			res.FailedChunkErrors = append(
				res.FailedChunkErrors,
				fmt.Errorf("chunk %d: %w", chunk.index, err),
			)
		}
	}

	return res, nil
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
