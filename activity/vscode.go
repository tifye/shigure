package activity

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
)

type VSCodeActivity struct {
	Workspace string `json:"workspace"`
	Filename  string `json:"fileName"`
	Language  string `json:"language"`
	Row       uint   `json:"row"`
	Col       uint   `json:"col"`
	CodeChunk string `json:"viewChunk"`
}

var defaultAcitivty = VSCodeActivity{
	Workspace: "La Soleil",
	Filename:  "Kitchen",
	Language:  "Meow",
	Row:       1,
	Col:       3,
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
}

func NewVSCodeActivityClient(logger *log.Logger) *VSCodeActivityClient {
	assert.AssertNotNil(logger)

	ac := &VSCodeActivityClient{
		logger:   logger,
		activity: defaultAcitivty,
	}

	ticker := time.NewTicker(5 * time.Minute)
	// intentionally run for lifetime
	go func() {
		for range ticker.C {
			ac.mu.RLock()
			if time.Since(ac.lastUpdate) > 30*time.Minute {
				ac.activity = defaultAcitivty
			}
			ac.mu.RUnlock()
		}
	}()

	return ac
}

func (c *VSCodeActivityClient) SetActivity(a VSCodeActivity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a.Filename = filepath.Base(a.Filename)
	c.activity = a
	c.lastUpdate = time.Now()
}

func (c *VSCodeActivityClient) Activity() VSCodeActivity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activity
}
