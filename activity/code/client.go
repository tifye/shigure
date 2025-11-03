package code

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/mux"
)

type VSCodeActivity struct {
	RepositoryURL string `json:"repository,omitempty"`
	Workspace     string `json:"workspace"`
	Filename      string `json:"fileName"`
	Language      string `json:"language"`
	Row           uint   `json:"row"`
	Col           uint   `json:"col"`
	CodeChunk     string `json:"viewChunk"`
}

var defaultAcitivty = VSCodeActivity{
	RepositoryURL: "https://github.com/tifye",
	Workspace:     "Unknown",
	Filename:      "inactive.md",
	Language:      "Probably english",
	Row:           1,
	Col:           3,
	CodeChunk: `



		if (Chocola && Vanilla) || Maple {
			// ヽ(*⌒▽⌒*)ﾉ
		}



	`,
}

type ActivityClient struct {
	logger     *log.Logger
	activity   VSCodeActivity
	lastUpdate atomic.Value
	mu         sync.RWMutex

	store         *CodeActivityStore
	redactedRepos *redactedRepos

	mux            *mux.Mux
	muxMessageType string
}

func NewActivityClient(
	logger *log.Logger,
	mux *mux.Mux,
	store *CodeActivityStore,
) *ActivityClient {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)
	assert.AssertNotNil(store)

	rr, err := newRedactedRepos("./data/redactedRepos")
	if err != nil {
		panic(err)
	}

	ac := &ActivityClient{
		logger:         logger,
		mux:            mux,
		muxMessageType: "vscode",
		activity:       defaultAcitivty,
		store:          store,
		redactedRepos:  rr,
		lastUpdate:     atomic.Value{},
	}
	ac.lastUpdate.Store(time.Now())

	ticker := time.NewTicker(5 * time.Minute)
	// intentionally run for lifetime
	go func() {
		for range ticker.C {
			if time.Since(ac.lastUpdate.Load().(time.Time)) >= 15*time.Minute {
				ac.mu.Lock()
				ac.activity = defaultAcitivty
				ac.mu.Unlock()
			}
		}
	}()

	return ac
}

func (c *ActivityClient) MessageType() string {
	return c.muxMessageType
}

func (c *ActivityClient) HandleMessage(_ *mux.Channel, _ []byte) error {
	return nil
}

func (c *ActivityClient) SetActivity(ctx context.Context, a VSCodeActivity) {
	c.logger.Debug("updating code activity", "repository", a.RepositoryURL)

	if c.redactedRepos.isRedacted(a.RepositoryURL) {
		a.CodeChunk = strings.Repeat("[REDACTED]\n", 10)
	}

	if a.Filename != "" {
		a.Filename = path.Base(strings.ReplaceAll(a.Filename, "\\", "/"))
		if a.Filename == "." || a.Filename == "/" || a.Filename == `\` {
			a.Filename = ""
		}
	}
	parts := strings.FieldsFunc(a.Filename, func(r rune) bool {
		return r == '\\' || r == '/'
	})
	if len(a.Filename) > 0 {
		a.Filename = parts[len(parts)-1]
	}

	c.mu.Lock()
	c.activity = a
	c.mu.Unlock()

	c.lastUpdate.Store(time.Now())

	err := c.store.Insert(ctx, CodeActivity{
		Repository: a.RepositoryURL,
		Workspace:  a.Workspace,
		Filename:   a.Filename,
		Language:   a.Language,
		Row:        a.Row,
		Col:        a.Col,
		CodeChunk:  a.CodeChunk,
		ReportedAt: time.Now(),
	})
	if err != nil {
		c.logger.Error("insert code activity", "err", err)
	}

	msgb, err := json.Marshal(a)
	if err != nil {
		c.logger.Error("marshal vscode activity", "err", err, "activity", a)
		return
	}
	c.mux.Broadcast(c.muxMessageType, msgb, nil)
}

func (c *ActivityClient) Activity() VSCodeActivity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activity
}

type Stats struct {
	TotalTimeSpent  string           `json:"totalTimeSpent"`
	LatestSessions  []SessionStat    `json:"latestSessions"`
	LanguageStats   []LanguageStat   `json:"languages"`
	RepositoryStats []RepositoryStat `json:"repositories"`
}

func (c *ActivityClient) CodeStats(ctx context.Context) (Stats, error) {
	totalTimeSpent, err := c.totalTimeSpent(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("total time spent: %s", err)
	}

	languageStats, err := c.languageStats(ctx, totalTimeSpent)
	if err != nil {
		return Stats{}, fmt.Errorf("language stats: %s", err)
	}

	repositoryStats, err := c.repositoryStats(ctx, totalTimeSpent)
	if err != nil {
		return Stats{}, fmt.Errorf("repository stats: %s", err)
	}

	sessionStats, err := c.sessionStats(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("session stats: %s", err)
	}

	return Stats{
		TotalTimeSpent:  totalTimeSpent.String(),
		LatestSessions:  sessionStats,
		LanguageStats:   languageStats,
		RepositoryStats: repositoryStats,
	}, nil
}

type RepositoryStat struct {
	Repository string  `json:"repository"`
	Percentage float64 `json:"percentage"`
	TimeSpent  string  `json:"timeSpent"`
}

func (c *ActivityClient) repositoryStats(ctx context.Context, totalTimeSpent time.Duration) ([]RepositoryStat, error) {
	assert.Assert(totalTimeSpent >= 0, "invalid total time spent")

	reports, err := c.store.RepositoryReports(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stored repository reports: %s", err)
	}

	if len(reports) == 0 {
		return nil, nil
	}

	stats := make([]RepositoryStat, len(reports))
	for i, report := range reports {
		timeSpent := time.Duration((report.OverallPercent / 100) * float64(totalTimeSpent))
		stats[i] = RepositoryStat{
			Repository: report.Repository,
			Percentage: math.Floor(report.OverallPercent*100) / 100,
			TimeSpent:  timeSpent.Truncate(time.Second).String(),
		}
	}

	return stats, nil
}

type LanguageStat struct {
	Language   string  `json:"language"`
	Percentage float64 `json:"percentage"`
	TimeSpent  string  `json:"timeSpent"`
}

func (c *ActivityClient) languageStats(ctx context.Context, totalTimeSpent time.Duration) ([]LanguageStat, error) {
	assert.Assert(totalTimeSpent >= 0, "invalid total time spent")

	reports, err := c.store.LanguagesReports(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stored language reports: %s", err)
	}

	if len(reports) == 0 {
		return nil, nil
	}

	stats := make([]LanguageStat, len(reports))
	for i, report := range reports {
		timeSpent := time.Duration((report.OverallPercent / 100) * float64(totalTimeSpent))
		stats[i] = LanguageStat{
			Language:   report.Language,
			Percentage: math.Floor(report.OverallPercent*100) / 100,
			TimeSpent:  timeSpent.Truncate(time.Second).String(),
		}
	}

	return stats, nil
}

type SessionStat struct {
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	Duration        string    `json:"duration"`
	TopRepositories []string  `json:"repositories"`
}

func (c *ActivityClient) sessionStats(ctx context.Context) ([]SessionStat, error) {
	sessions, err := c.store.Sessions(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("get stored sessions: %s", err)
	}

	if len(sessions) == 0 {
		return nil, nil
	}

	stats := make([]SessionStat, len(sessions))
	for i, session := range sessions {
		repositories := make([]string, len(session.TopRepositories))
		for j, repo := range session.TopRepositories {
			switch v := repo.(type) {
			case string:
				repositories[j] = v
			case []byte:
				repositories[j] = string(v)
			default:
				repositories[j] = fmt.Sprint(v)
			}
		}
		stats[i] = SessionStat{
			Start: session.Start,
			End:   session.End,
			Duration: session.End.Sub(session.Start).
				Truncate(time.Second).
				String(),
			TopRepositories: repositories,
		}
	}

	return stats, nil
}

func (c *ActivityClient) totalTimeSpent(ctx context.Context) (time.Duration, error) {
	ts, err := c.store.TotatTimeSpent(ctx)
	if err != nil {
		return 0, fmt.Errorf("get total hours: %w", err)
	}

	return time.Duration(time.Second * time.Duration(ts.Seconds)).Truncate(time.Second), nil
}
