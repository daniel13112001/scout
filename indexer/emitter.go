package indexer

import (
	"database/sql"
	"encoding/binary"
	"math"
)

// EmbeddingRecord is a single chunk's embedding, ready to be persisted.
type EmbeddingRecord struct {
	FilePath   string
	ChunkIndex int
	Content    string
	Embedding  []float32
}

// Emitter accepts embedding records for persistence. Implementations must
// be safe for concurrent calls to Emit from multiple goroutines.
type Emitter interface {
	Emit(record EmbeddingRecord) error
}

// dbWriter is the single writer to the database. File workers call Emit
// concurrently, which just queues the record on a channel; one background
// goroutine drains that channel and performs all the db.Exec calls, so the
// database only ever sees a single writer at a time.
type dbWriter struct {
	db      *sql.DB
	records chan EmbeddingRecord
	done    chan struct{}
	errCh   chan error
}

func newDBWriter(db *sql.DB, bufferSize int) *dbWriter {
	return &dbWriter{
		db:      db,
		records: make(chan EmbeddingRecord, bufferSize),
		done:    make(chan struct{}),
		errCh:   make(chan error, 1),
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
func (w *dbWriter) Emit(record EmbeddingRecord) error {
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

func (w *dbWriter) write(record EmbeddingRecord) error {
	blob := encodeEmbedding(record.Embedding)

	_, err := w.db.Exec(
		`INSERT INTO chunks (file_path, chunk_index, content, embedding) VALUES (?, ?, ?, ?)`,
		record.FilePath, record.ChunkIndex, record.Content, blob,
	)
	return err
}

func encodeEmbedding(embedding []float32) []byte {
	buf := make([]byte, 4*len(embedding))
	for i, f := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}
