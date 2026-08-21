package indexer

import (
	"database/sql"
	"fmt"

	// Importing this package registers a build of SQLite (via the ncruces
	// driver) with the sqlite-vec extension compiled in, and provides
	// SerializeFloat32 for encoding embeddings into the BLOB format it
	// expects.
	vecembed "github.com/asg017/sqlite-vec-go-bindings/ncruces"
)

// ChunkRecord is a single chunk, embedded and ready to be persisted.
type ChunkRecord struct {
	ChunkIndex     int
	Content        string
	ContentHash    []byte
	StartLine      int
	EndLine        int
	EmbeddingModel string
	Embedding      []float32
}

// FileRecord is one file's metadata plus a batch of its embedded chunks,
// persisted together as a single write. A file with no chunks (e.g. empty)
// is still emitted with an empty Chunks slice, so it's still recorded in
// files. A large file is split across multiple FileRecords sharing the
// same Path, so no single write has to hold an entire file's chunks in
// memory or in one transaction.
type FileRecord struct {
	Path       string
	ModifiedAt int64
	FileHash   []byte

	Chunks []ChunkRecord
}

// Emitter accepts file records for persistence. Implementations must be
// safe for concurrent calls to Emit from multiple goroutines.
type Emitter interface {
	Emit(record FileRecord) error
}

// dbWriter is the single writer to the database. File workers call Emit
// concurrently, which just queues the record on a channel; one background
// goroutine drains that channel and performs all the db.Exec calls, so the
// database only ever sees a single writer at a time.
type dbWriter struct {
	db      *sql.DB
	records chan FileRecord
	done    chan struct{}
	errCh   chan error

	// Tracks which files' old chunks have already been cleared this run,
	// keyed by files.id. A file can arrive across multiple FileRecords (see
	// FileRecord), so clearing has to happen exactly once, on the first
	// record seen for that file, not once per record.
	clearedFiles map[int64]bool
}

func newDBWriter(db *sql.DB, bufferSize int) *dbWriter {
	return &dbWriter{
		db:           db,
		records:      make(chan FileRecord, bufferSize),
		done:         make(chan struct{}),
		errCh:        make(chan error, 1),
		clearedFiles: make(map[int64]bool),
	}
}

// start launches the single writer goroutine. Must be called before Emit.
func (w *dbWriter) start() {
	go func() {
		defer close(w.done)

		for record := range w.records {
			if err := w.write(record); err != nil {
				select {
				case w.errCh <- err:
				default:
				}
			}
		}
	}()
}

// Emit queues a record for the writer goroutine. Safe to call concurrently.
func (w *dbWriter) Emit(record FileRecord) error {
	w.records <- record
	return nil
}

// close stops accepting new records, waits for the writer goroutine to
// drain the queue, and returns the first write error encountered, if any.
func (w *dbWriter) close() error {
	close(w.records)
	<-w.done

	select {
	case err := <-w.errCh:
		return err
	default:
		return nil
	}
}

func (w *dbWriter) write(record FileRecord) error {
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert on path so reindexing the same file doesn't create duplicate
	// files rows; harmless to repeat across multiple records for the same
	// file, since it's idempotent.
	var fileID int64
	err = tx.QueryRow(
		`INSERT INTO files (path, modified_at, file_hash, status, last_error)
		 VALUES (?, ?, ?, 'ok', NULL)
		 ON CONFLICT(path) DO UPDATE SET
		   modified_at = excluded.modified_at,
		   file_hash = excluded.file_hash,
		   status = 'ok',
		   last_error = NULL
		 RETURNING id`,
		record.Path, record.ModifiedAt, record.FileHash,
	).Scan(&fileID)
	if err != nil {
		return fmt.Errorf("upserting file: %w", err)
	}

	if !w.clearedFiles[fileID] {
		// vec_chunks has no real foreign key to chunks (sqlite-vec's vec0
		// tables don't support them), so its rows must be deleted
		// explicitly here rather than relying on ON DELETE CASCADE.
		if _, err := tx.Exec(`DELETE FROM vec_chunks WHERE rowid IN (SELECT id FROM chunks WHERE file_id = ?)`, fileID); err != nil {
			return fmt.Errorf("clearing old vectors: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM chunks WHERE file_id = ?`, fileID); err != nil {
			return fmt.Errorf("clearing old chunks: %w", err)
		}
		w.clearedFiles[fileID] = true
	}

	for _, chunk := range record.Chunks {
		res, err := tx.Exec(
			`INSERT INTO chunks (file_id, chunk_index, content, content_hash, start_line, end_line, embedding_model)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fileID, chunk.ChunkIndex, chunk.Content, chunk.ContentHash,
			chunk.StartLine, chunk.EndLine, chunk.EmbeddingModel,
		)
		if err != nil {
			return fmt.Errorf("inserting chunk %d: %w", chunk.ChunkIndex, err)
		}

		chunkID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("reading chunk id: %w", err)
		}

		embeddingBlob, err := vecembed.SerializeFloat32(chunk.Embedding)
		if err != nil {
			return fmt.Errorf("serializing embedding: %w", err)
		}

		if _, err := tx.Exec(`INSERT INTO vec_chunks (rowid, embedding) VALUES (?, ?)`, chunkID, embeddingBlob); err != nil {
			return fmt.Errorf("inserting vector for chunk %d: %w", chunk.ChunkIndex, err)
		}
	}

	return tx.Commit()
}
