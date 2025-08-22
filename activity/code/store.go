package code

import (
	"context"
	"time"

	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/storage"
)

type CodeActivity struct {
	Repository string    `db:"repository"`
	Workspace  string    `db:"workspace"`
	Filename   string    `db:"filename"`
	Language   string    `db:"language"`
	Row        uint      `db:"row"`
	Col        uint      `db:"column"`
	CodeChunk  string    `db:"chunk"`
	ReportedAt time.Time `db:"reported_at"`
}

type CodeActivityStore struct {
	db storage.DuckDB
}

func NewCodeActivityStore(db storage.DuckDB) *CodeActivityStore {
	assert.AssertNotNil(db)
	return &CodeActivityStore{
		db: db,
	}
}

func (s *CodeActivityStore) Insert(ctx context.Context, ca CodeActivity) error {
	query := `
	INSERT INTO code_activity (
		repository,
		workspace,
		filename,
		language,
		"row",
		"column",
		code_chunk,
		reported_at
	)
	VALUES (?,?,?,?,?,?,?,?,)
	`
	_, err := s.db.ExecContext(
		ctx, query,
		ca.Repository,
		ca.Workspace,
		ca.Filename,
		ca.Language,
		ca.Row,
		ca.Col,
		ca.CodeChunk,
		ca.ReportedAt,
	)
	return err
}
