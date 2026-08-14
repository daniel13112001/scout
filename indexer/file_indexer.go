package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"time"

	"path/filepath"

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
			e := fi.processFile(path)
			if e != nil {
				errCh <- e
			}
		}()

		return nil
	})

	go func() {
	wg.Wait()
	close(errCh)
	}()


	for err := range errCh{
		return err
	}
	return nil
}

func (fi *FileIndexer) processFile(path string) error {
	time.Sleep(1000 * time.Millisecond)
	fmt.Println(path)
	return nil
}
