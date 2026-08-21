package app

import (
	"github.com/daniel13112001/scout/indexer"
	"github.com/daniel13112001/scout/search"
)

type Dependencies struct {
	FileIndexer *indexer.FileIndexer
	Searcher    *search.Searcher
}
