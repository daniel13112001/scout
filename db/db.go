package db

import (
    "database/sql"
    _ "embed"
)


//go:embed schema.sql
var schema string

func InitDB(database *sql.DB) error {

    _, err := database.Exec(schema)
    return err
}