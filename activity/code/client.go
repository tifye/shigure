package code

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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

	store *CodeActivityStore

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

	ac := &ActivityClient{
		logger:         logger,
		mux:            mux,
		muxMessageType: "vscode",
		activity:       defaultAcitivty,
		store:          store,
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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("updating code activity", "repository", a.RepositoryURL)

	parts := strings.FieldsFunc(a.Filename, func(r rune) bool {
		return r == '\\' || r == '/'
	})
	if len(a.Filename) > 0 {
		a.Filename = parts[len(parts)-1]
	}
	c.activity = a
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
	LanguageStats []LanguageStat `json:"languages"`
}

func (c *ActivityClient) CodeStats(ctx context.Context) (Stats, error) {
	languageStats, err := c.languageStats(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("language stats: %s", err)
	}

	return Stats{
		LanguageStats: languageStats,
	}, nil
}

type LanguageStat struct {
	Language      string    `json:"language"`
	Percentage    uint      `json:"percentage"`
	TimesReported uint      `json:"timesReported"`
	HoursSpent    float64   `json:"hoursSpent"`
	MinutesSpent  float64   `json:"minutesSpent"`
	SecondsSpent  float64   `json:"secondsSpent"`
	LastUsed      time.Time `json:"lastUsed"`
}

func (c *ActivityClient) languageStats(ctx context.Context) ([]LanguageStat, error) {
	reports, err := c.store.LanguagesReports(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting language reports: %s", err)
	}

	var totalReports uint
	for _, report := range reports {
		totalReports = totalReports + report.TimesReported
	}

	stats := make([]LanguageStat, len(reports))
	for i, report := range reports {
		percent := math.Round(float64(report.TimesReported) / float64(totalReports) * 100)

		// For now reports are generated on a 2sec interval.
		// In the future we want to use DuckDB for this.
		timeSpent := time.Duration(time.Second * 2 * time.Duration(report.TimesReported))
		stats[i] = LanguageStat{
			Language:      report.Language,
			Percentage:    uint(percent),
			TimesReported: report.TimesReported,
			HoursSpent:    timeSpent.Hours(),
			MinutesSpent:  timeSpent.Minutes(),
			SecondsSpent:  timeSpent.Seconds(),
			LastUsed:      report.LastReported,
		}
	}

	return stats, nil
}
