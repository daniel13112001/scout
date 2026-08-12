package indexer

import (
	"database/sql"
	"errors"

	"github.com/daniel13112001/scout/embedder"
)

type FileIndexer struct {
	db       *sql.DB
	embedder embedder.Embedder
}

func NewFileIndexer(db *sql.DB, embedder embedder.Embedder) (*FileIndexer, error) {

	if db == nil {
		return nil, errors.New("db is nil. cannot initialize file indexer with valid db.")
	}
	return &FileIndexer{
		db:       db,
		embedder: embedder,
	}, nil

}

func (*FileIndexer) Index(dir string, extensions string) error {
	return nil
}
