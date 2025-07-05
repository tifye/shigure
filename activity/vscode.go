package activity

import (
	"strings"
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
	logger      *log.Logger
	activity    VSCodeActivity
	lastUpdate  time.Time
	mu          sync.RWMutex
	subscribers map[chan<- VSCodeActivity]struct{}
	regch       chan chan<- VSCodeActivity
	unregch     chan chan<- VSCodeActivity
	notify      chan VSCodeActivity
}

func NewVSCodeActivityClient(logger *log.Logger) *VSCodeActivityClient {
	assert.AssertNotNil(logger)

	ac := &VSCodeActivityClient{
		logger:      logger,
		activity:    defaultAcitivty,
		subscribers: map[chan<- VSCodeActivity]struct{}{},
		regch:       make(chan chan<- VSCodeActivity),
		unregch:     make(chan chan<- VSCodeActivity),
		notify:      make(chan VSCodeActivity),
	}

	ticker := time.NewTicker(5 * time.Minute)
	// intentionally run for lifetime
	go func() {
		for {
			select {
			case <-ticker.C:
				ac.mu.RLock()
				if time.Since(ac.lastUpdate) > 30*time.Minute {
					ac.activity = defaultAcitivty
				}
				ac.mu.RUnlock()
			case sub := <-ac.regch:
				ac.subscribers[sub] = struct{}{}
				ac.logger.Info("VSC Activity subscriber added")
			case sub := <-ac.unregch:
				close(sub)
				delete(ac.subscribers, sub)
				ac.logger.Info("VSC Activity subscriber removed")
			case a := <-ac.notify:
				ac.logger.Debug("notifying subscribers")
				for sub := range ac.subscribers {
					sub <- a
				}
			}
		}
	}()

	return ac
}

func (r *VSCodeActivityClient) Subscribe(sub chan<- VSCodeActivity) {
	r.regch <- sub
}

func (r *VSCodeActivityClient) Unsubscribe(sub chan<- VSCodeActivity) {
	r.regch <- sub
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

	c.notify <- a
}

func (c *VSCodeActivityClient) Activity() VSCodeActivity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activity
}
