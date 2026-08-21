package indexer

import (
	"crypto/sha256"
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
	chunkSize = 500

	// writeGroupSize bounds how many chunks of one file are embedded and
	// written together. Keeping this bounded (rather than processing a
	// whole file as one unit) keeps memory and per-transaction size
	// independent of file size, and means a failure partway through a huge
	// file doesn't lose everything already embedded.
	writeGroupSize = 32

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
				res, err := fi.processFile(path, writer)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					continue
				}

				if len(res.FailedChunkErrors) > 0 {
					select {
					case errCh <- fmt.Errorf("%s: %w", path, errors.Join(res.FailedChunkErrors...)):
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

// processFile reads and chunks a file, then embeds and emits its chunks in
// bounded groups (see writeGroupSize) for the single db writer to persist.
// A file with no chunks is still emitted once, empty, so it's still
// recorded in files.
func (fi *FileIndexer) processFile(path string, emitter Emitter) (ProcessFileResult, error) {
	content, err := ReadFile(path)
	if err != nil {
		return ProcessFileResult{}, err
	}

	chunks, err := ChunkText(content, chunkSize)
	if err != nil {
		return ProcessFileResult{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return ProcessFileResult{}, err
	}
	modifiedAt := info.ModTime().Unix()

	fileHashSum := sha256.Sum256([]byte(content))
	fileHash := fileHashSum[:]

	res := ProcessFileResult{}

	if len(chunks) == 0 {
		if err := emitter.Emit(FileRecord{Path: path, ModifiedAt: modifiedAt, FileHash: fileHash}); err != nil {
			res.FailedChunkErrors = append(res.FailedChunkErrors, fmt.Errorf("recording empty file: %w", err))
		}
		return res, nil
	}

	modelID := fi.Embedder.ModelID()

	for start := 0; start < len(chunks); start += writeGroupSize {
		end := min(start+writeGroupSize, len(chunks))
		group := chunks[start:end]

		texts := make([]string, len(group))
		for i, chunk := range group {
			texts[i] = chunk.content
		}

		embeddings, err := fi.Embedder.Embed(texts)
		if err != nil {
			res.FailedChunkErrors = append(
				res.FailedChunkErrors,
				fmt.Errorf("embedding chunks %d-%d: %w", group[0].index, group[len(group)-1].index, err),
			)
			continue
		}

		chunkRecords := make([]ChunkRecord, len(group))
		for i, chunk := range group {
			contentHashSum := sha256.Sum256([]byte(chunk.content))

			chunkRecords[i] = ChunkRecord{
				ChunkIndex:     chunk.index,
				Content:        chunk.content,
				ContentHash:    contentHashSum[:],
				StartLine:      chunk.startLine,
				EndLine:        chunk.endLine,
				EmbeddingModel: modelID,
				Embedding:      embeddings[i],
			}
		}

		if err := emitter.Emit(FileRecord{
			Path:       path,
			ModifiedAt: modifiedAt,
			FileHash:   fileHash,
			Chunks:     chunkRecords,
		}); err != nil {
			res.FailedChunkErrors = append(
				res.FailedChunkErrors,
				fmt.Errorf("writing chunks %d-%d: %w", group[0].index, group[len(group)-1].index, err),
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
