package storage

import (
	_ "embed"

	"github.com/jmoiron/sqlx"
	_ "github.com/marcboeker/go-duckdb/v2"
)

//go:embed schema/codeActivity.sql
var codeActivitySchema []byte

type DuckDB = *sqlx.DB

func InitDuckDB() (DuckDB, error) {
	db, err := sqlx.Connect("duckdb", "./data/analytics.db")
	if err != nil {
		return nil, err
	}

	_ = db.MustExec(string(codeActivitySchema))

	return db, nil
}
