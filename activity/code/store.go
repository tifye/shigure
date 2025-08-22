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
	insert into code_activity (
		repository,
		workspace,
		filename,
		language,
		"row",
		"column",
		code_chunk,
		reported_at
	)
	values (?,?,?,?,?,?,?,?)
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

type StoredLanguageReport struct {
	Language      string    `db:"language"`
	TimesReported uint      `db:"times_reported"`
	LastReported  time.Time `db:"last_reported"`
}

func (s *CodeActivityStore) LanguagesReports(ctx context.Context) ([]StoredLanguageReport, error) {
	query := `
	select language, 
		count(*) as times_reported,
		max(reported_at) as last_reported
	from code_activity
	group by language
	order by times_reported desc;
	`
	var reports []StoredLanguageReport
	err := s.db.SelectContext(ctx, &reports, query)
	return reports, err
}

type StoredSession struct {
	// ID is not guaranteed to be consistent,
	// it is generated on a per query basis.
	ID    uint      `db:"id"`
	Start time.Time `db:"start"`
	End   time.Time `db:"end"`
	// String slice of the top most used
	// repositories during a given session in order of
	// most -> least used.
	TopRepositories []any `db:"top_repositories"`
}

func (s *CodeActivityStore) Sessions(ctx context.Context, limit uint) ([]StoredSession, error) {
	assert.Assert(limit < 100, "limit too large")

	query := `
	select id, "start", "end", top_repositories from sessions
	limit ?
	`
	var sessions []StoredSession
	err := s.db.SelectContext(ctx, &sessions, query, limit)
	return sessions, err
}

type StoredTimeSpent struct {
	Seconds float64 `db:"seconds"`
	Minutes float64 `db:"minutes"`
	Hours   float64 `db:"hours"`
	Days    float64 `db:"days"`
	Weeks   float64 `db:"weeks"`
}

func (s *CodeActivityStore) TotalHours(ctx context.Context) (StoredTimeSpent, error) {
	query := `
	with totalSeconds as (
		select sum(epoch(duration)) as seconds from sessions
	)
	select 
		seconds,
		seconds/60 as minutes,
		minutes/60 as hours,
		hours/24 as days,
		days/7 as weeks
	from totalSeconds
	`
	row := s.db.QueryRowxContext(ctx, query)
	var timeSpent StoredTimeSpent
	err := row.StructScan(&timeSpent)
	return timeSpent, err
}
