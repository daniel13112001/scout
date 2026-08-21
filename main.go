package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/daniel13112001/scout/app"
	"github.com/daniel13112001/scout/cli"
	"github.com/daniel13112001/scout/commands"
	"github.com/daniel13112001/scout/config"
	scoutdb "github.com/daniel13112001/scout/db"
	"github.com/daniel13112001/scout/embedder"
	"github.com/daniel13112001/scout/indexer"
	"github.com/daniel13112001/scout/search"

	"database/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func main() {

	args := os.Args

	commandMap := map[string]commands.Command{
		"find":   commands.Find,
		"index":  commands.Index,
		"help":   commands.Help,
		"sync":   commands.Sync,
		"config": commands.Config,
	}

	if len(args) < 2 {

		usage := `
		Usage:
		scout <command> [arguments]

		Commands:
		find       Search indexed files
		index      Index files
		sync       Synchronize the index
		help       Show help

		`
		fmt.Println(usage)

		return

	}

	cmd, ok := commandMap[args[1]]

	if !ok {
		fmt.Printf("Unrecognized command. %v is not a valid command\n", args[1])
		return
	}

	cfg, err := config.Load()

	if err != nil {
		fmt.Println("unable to load config: ", err)
		return
	}

	logFile, err := config.OpenLog()

	if err != nil {
		fmt.Println("unable to open log file: ", err)
		return
	}

	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	db, err := sql.Open("sqlite3", cfg.DB.Path)

	if err != nil {
		fmt.Println("unable to initialize database: ", err)
		return
	}

	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Println("database unavailable. shutting down: ", err)
		return
	}

	//db.SetMaxOpenConns(1)
	err = scoutdb.InitDB(db)

	if err != nil {
		fmt.Println("unable to execute db initialization schema: %w ", err)
		return
	}

	embedderLoadStart := time.Now()

	localEmbedder, err := embedder.NewLocalEmbedder(embedder.EmbedderConfig{
		ModelPath:      cfg.Embedder.ModelPath,
		TokenizerPath:  cfg.Embedder.TokenizerPath,
		OrtLibraryPath: cfg.Embedder.OrtLibraryPath,
		BatchSize:      cfg.Embedder.BatchSize,
	})

	if err != nil {
		fmt.Println("unable to load embedding model: ", err)
		return
	}

	defer localEmbedder.Close()

	logger.Printf("embedder loaded in %s", time.Since(embedderLoadStart))

	searcher, err := search.NewSearcher(db, localEmbedder, logger)

	if err != nil {
		fmt.Println("unable to initialize searcher: ", err)
		return
	}

	// Construct application dependencies
	ctx := context.Background()
	deps := app.Dependencies{
		FileIndexer: &indexer.FileIndexer{Db: db, Embedder: localEmbedder, IndexConfig: cfg.Index, Logger: logger},
		Searcher:    searcher,
		Logger:      logger,
	}
	parsedArgs, err := cli.Parse(args[2:])

	if err != nil {
		fmt.Println(err)
		return
	}

	err = cmd(ctx, parsedArgs, deps)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%v run completed", args[1])

}
