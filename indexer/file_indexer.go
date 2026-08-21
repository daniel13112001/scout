package indexer

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/daniel13112001/scout/config"
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
	Db          *sql.DB
	Embedder    embedder.Embedder
	IndexConfig config.IndexConfig
}

type ProcessFileResult struct {
	FailedChunkErrors []error
}

func NewFileIndexer(db *sql.DB, embedder embedder.Embedder, indexConfig config.IndexConfig) (*FileIndexer, error) {
	if db == nil {
		return nil, errors.New("db is nil. cannot initialize file indexer")
	}

	if embedder == nil {
		return nil, errors.New("embedder is nil. cannot initialize file indexer")
	}

	return &FileIndexer{
		Db:          db,
		Embedder:    embedder,
		IndexConfig: indexConfig,
	}, nil
}

// shouldSkipDir reports whether a directory (by name) should be pruned
// from the walk entirely, per IndexConfig.IgnoreDirs.
func (fi *FileIndexer) shouldSkipDir(name string) bool {
	return slices.Contains(fi.IndexConfig.IgnoreDirs, name)
}

// shouldIndexFile reports whether a file (by name) passes
// IndexConfig.IgnorePatterns and IndexConfig.AllowedExtensions.
func (fi *FileIndexer) shouldIndexFile(name string) bool {
	for _, pattern := range fi.IndexConfig.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return false
		}
	}

	ext := strings.ToLower(filepath.Ext(name))
	for _, allowed := range fi.IndexConfig.AllowedExtensions {
		if ext == strings.ToLower(allowed) {
			return true
		}
	}

	return false
}

// maxFileSizeBytes returns the configured max file size in bytes, or 0 if
// MaxFileSizeMB is unset, meaning no limit.
func (fi *FileIndexer) maxFileSizeBytes() int64 {
	return int64(fi.IndexConfig.MaxFileSizeMB) * 1024 * 1024
}

// IndexDirectory walks a directory and processes each file, skipping
// directories and files per IndexConfig. If recursive is false, only files
// directly in dir are processed - subdirectories are not descended into.
func (fi *FileIndexer) IndexDirectory(dir string, recursive bool) error {
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
			if path != dir {
				if !recursive {
					return filepath.SkipDir
				}
				if fi.shouldSkipDir(d.Name()) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if !fi.shouldIndexFile(d.Name()) {
			return nil
		}

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

// processFile skips path entirely if it's too large or unchanged since the
// last index (see maxFileSizeBytes and isUnchanged); otherwise it reads and
// chunks it, then embeds and emits its chunks in bounded groups (see
// writeGroupSize) for the single db writer to persist. A file with no
// chunks is still emitted once, empty, so it's still recorded in files.
func (fi *FileIndexer) processFile(path string, emitter Emitter) (ProcessFileResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ProcessFileResult{}, err
	}

	if maxSize := fi.maxFileSizeBytes(); maxSize > 0 && info.Size() > maxSize {
		return ProcessFileResult{}, nil
	}

	modifiedAt := info.ModTime().Unix()

	unchanged, err := fi.isUnchanged(path, modifiedAt)
	if err != nil {
		return ProcessFileResult{}, err
	}
	if unchanged {
		return ProcessFileResult{}, nil
	}

	content, err := ReadFile(path)
	if err != nil {
		return ProcessFileResult{}, err
	}

	chunks, err := ChunkText(content, chunkSize)
	if err != nil {
		return ProcessFileResult{}, err
	}

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

// isUnchanged reports whether path is already indexed with this exact
// modification time. This is a plain read, safe to run concurrently across
// file workers - only writes need to go through the single dbWriter.
func (fi *FileIndexer) isUnchanged(path string, modifiedAt int64) (bool, error) {
	var stored int64

	err := fi.Db.QueryRow(`SELECT modified_at FROM files WHERE path = ?`, path).Scan(&stored)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return stored == modifiedAt, nil
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
