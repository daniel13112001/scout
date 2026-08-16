package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"path/filepath"

	"github.com/daniel13112001/scout/embedder"
)

type FileIndexer struct {
	Db       *sql.DB
	Embedder embedder.Embedder
}

type ProcessFileResult struct {
	isCompleteSuccess bool
	isPartialSuccess  bool
	isCompleteFailure bool
	failedChunkErrors []error
}

func NewFileIndexer(db *sql.DB, embedder embedder.Embedder) (*FileIndexer, error) {

	if db == nil {
		return nil, errors.New("db is nil. cannot initialize file indexer with valid db.")
	}
	return &FileIndexer{
		Db:       db,
		Embedder: embedder,
	}, nil

}

// The file indexer receives a path to a directory
// Its job is to recursively walk the directory, embed every file
// and write the file with metadata to the db.

func (fi *FileIndexer) IndexDirectory(dir string, extensions string) error {

	// walk down the directory
	// parse documents
	// chunk documents
	// embed documents
	// save embeddings and metadata to db

	var wg sync.WaitGroup
	errCh := make(chan error)

	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// TODO add logic for filtering by allowed extensions

		wg.Add(1)
		go func() {
			defer wg.Done()
			_, p := fi.processFile(path)
			if p != nil {
				errCh <- p
			}
		}()

		return nil
	})

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		return err
	}
	return nil
}

// ProcessFile takes the path to a file, reads and chunks its contents,
// embeds each chunk, and writes the resulting embeddings and metadata
// to the database.
//
// A failure that prevents the file from being processed at all, such as
// being unable to read the file or chunk its contents, is returned as a
// standard error.
//
// Individual chunk failures do not stop processing the remaining chunks.
// Instead, the errors are recorded in ProcessFileResult.failedChunkErrors.
// This allows the file to be partially indexed even if some chunks fail.
//
// On complete success, ProcessFileResult.isCompleteSuccess is true.
// If one or more chunks fail but the remaining chunks are processed,
// ProcessFileResult.isPartialSuccess is true.
func (fi *FileIndexer) processFile(path string) (ProcessFileResult, error) {

	res := ProcessFileResult{}

	content, err := ReadFile(path)
	if err != nil {
		res.isCompleteFailure = true
		return res, err
	}

	fileChunks, err := ChunkText(content, 500)
	if err != nil {
		res.isCompleteFailure = true
		return res, err
	}

	for _, chunk := range fileChunks {
		_, err := fi.Embedder.Embed(chunk.content)
		if err != nil {
			res.failedChunkErrors = append(
				res.failedChunkErrors,
				fmt.Errorf("chunk %d: %w", chunk.index, err),
			)
			continue
		}

		_, err = fi.Db.Exec(`
			INSERT INTO files (path, modified_at)
			VALUES (?, ?)
		`, "dummy.txt", 0)

		if err != nil {
			return res, err
		}

		// Store embedding in DB...
	}

	if len(res.failedChunkErrors) == 0 {
		res.isCompleteSuccess = true
	} else {
		res.isPartialSuccess = true
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
