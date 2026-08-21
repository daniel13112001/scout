package main

import (
	"context"
	"fmt"
	"os"

	"github.com/daniel13112001/scout/app"
	"github.com/daniel13112001/scout/cli"
	"github.com/daniel13112001/scout/commands"
	scoutdb "github.com/daniel13112001/scout/db"
	"github.com/daniel13112001/scout/embedder"
	"github.com/daniel13112001/scout/indexer"

	"database/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
)

const (
	modelPath      = "models/model_qint8_avx512_vnni.onnx"
	tokenizerPath  = "models/tokenizer.json"
	ortLibraryPath = "third_party/onnxruntime/libonnxruntime.dylib"
	embedBatchSize = 32
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

	db, err := sql.Open("sqlite3", "scout.db")

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

	localEmbedder, err := embedder.NewLocalEmbedder(embedder.EmbedderConfig{
		ModelPath:      modelPath,
		TokenizerPath:  tokenizerPath,
		OrtLibraryPath: ortLibraryPath,
		BatchSize:      embedBatchSize,
	})

	if err != nil {
		fmt.Println("unable to load embedding model: ", err)
		return
	}

	defer localEmbedder.Close()

	// Construct application dependencies
	ctx := context.Background()
	deps := app.Dependencies{
		FileIndexer: &indexer.FileIndexer{Db: db, Embedder: localEmbedder},
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
	fmt.Printf("%v run completed", cmd)

}
