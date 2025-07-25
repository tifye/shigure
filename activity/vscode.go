package activity

import (
	"encoding/json"
	"strings"
	"sync"
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

type VSCodeActivityClient struct {
	logger     *log.Logger
	activity   VSCodeActivity
	lastUpdate time.Time
	mu         sync.RWMutex

	mux            *mux.Mux
	muxMessageType string
}

func NewVSCodeActivityClient(logger *log.Logger, mux *mux.Mux) *VSCodeActivityClient {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)

	ac := &VSCodeActivityClient{
		logger:         logger,
		mux:            mux,
		muxMessageType: "vscode",
		activity:       defaultAcitivty,
	}

	ticker := time.NewTicker(5 * time.Minute)
	// intentionally run for lifetime
	go func() {
		for range ticker.C {
			if time.Since(ac.lastUpdate) >= 15*time.Minute {
				ac.mu.Lock()
				ac.activity = defaultAcitivty
				ac.mu.Unlock()
			}
		}
	}()

	return ac
}

func (c *VSCodeActivityClient) MessageType() string {
	return c.muxMessageType
}

func (c *VSCodeActivityClient) HandleMessage(_ *mux.Channel, _ []byte) error {
	return nil
}

func (c *VSCodeActivityClient) SetActivity(a VSCodeActivity) {
	c.mu.Lock()
	defer c.mu.Unlock()

	parts := strings.FieldsFunc(a.Filename, func(r rune) bool {
		return r == '\\' || r == '/'
	})
	a.Filename = parts[len(parts)-1]
	c.activity = a
	c.lastUpdate = time.Now()

	c.logger.Debug(a.RepositoryURL)

	msgb, err := json.Marshal(a)
	if err != nil {
		c.logger.Error("marshal vscode activity", "err", err, "activity", a)
		return
	}

	c.mux.Broadcast(c.muxMessageType, msgb, nil)
}

func (c *VSCodeActivityClient) Activity() VSCodeActivity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activity
}
