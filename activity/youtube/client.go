package youtube

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/charmbracelet/log"

	"github.com/tifye/shigure/assert"
)

//go:embed templates/*
var templates embed.FS

type Activity struct {
	Id           string
	Title        string
	Author       string
	Url          string
	ThumbnailUrl string
	Duration     time.Duration
}

type ActivityClient struct {
	apiKey string
	logger *log.Logger

	lastUpdate      atomic.Value
	currentActivity Activity
	dirty           atomic.Bool
	mu              sync.RWMutex
	fileMu          sync.Mutex
}

func NewClient(logger *log.Logger, apiKey string) *ActivityClient {
	assert.AssertNotNil(logger)
	assert.AssertNotEmpty(apiKey)

	err := os.MkdirAll("data", 0644)
	if err != nil {
		panic(err)
	}

	client := &ActivityClient{
		logger:     logger,
		apiKey:     apiKey,
		lastUpdate: atomic.Value{},
	}
	client.lastUpdate.Store(time.Now())

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			a := client.Activity()
			if time.Since(client.lastUpdate.Load().(time.Time)) >= a.Duration {
				client.ClearActivity()
			}
		}
	}()

	client.ClearActivity()
	return client
}

func (c *ActivityClient) SetYoutubeActivity(ctx context.Context, videoId string) error {
	assert.AssertNotEmpty(videoId)

	if c.Activity().Id == videoId {
		return nil
	}

	resource, err := FetchYoutubeVideoResource(ctx, c.logger, c.apiKey, videoId)
	if err != nil {
		return fmt.Errorf("fetch video: %s", err)
	}

	if !isYoutubeVideoDurationLessThan1Day(resource.ContentDetails.Duration) {
		c.logger.Info("skipping video 24h+ duration video", "videoId", resource.Id, "title", resource.Snippet.Title)
		return nil
	}

	c.setActivity(Activity{
		Id:           resource.Id,
		Url:          fmt.Sprintf("https://www.youtube.com/watch?v=%s", resource.Id),
		Title:        resource.Snippet.Title,
		Author:       resource.Snippet.ChannelTitle,
		ThumbnailUrl: resource.Snippet.Thumbnails.HighRes.Url,
		Duration:     parseYoutubeVideoDuration(resource.ContentDetails.Duration),
	})

	return nil
}

// Returned true/false whether the duration if less
// than 1 day. Duration format follows ISO_8601.
// See https://en.wikipedia.org/wiki/ISO_8601#Durations
func isYoutubeVideoDurationLessThan1Day(duration string) bool {
	return strings.HasPrefix(duration, "PT")
}

// Converts an ISO_8601 duration into a time.Duration.
// Only accepts durations less than 1 day.
// See https://en.wikipedia.org/wiki/ISO_8601#Durations
func parseYoutubeVideoDuration(duration string) time.Duration {
	// PT##H##M##S
	assert.Assert(len(duration) <= 11, fmt.Sprintf("invalid duration string: %s", duration))
	// PT+#T
	assert.Assert(len(duration) >= 4, fmt.Sprintf("invalid duration string: %s", duration))
	assert.Assert(strings.HasPrefix(duration, "PT"), "expected duration string to have 'PT' ISO_8601 prefix")

	duration = strings.TrimPrefix(duration, "PT")
	var (
		hourPart    string
		minutesPart string
		secondsPart string
	)
	start := 0
	for pos := range len(duration) {
		switch duration[pos] {
		case 'H':
			hourPart = duration[start:pos]
			start = pos + 1
		case 'M':
			minutesPart = duration[start:pos]
			start = pos + 1
		case 'S':
			secondsPart = duration[start:pos]
			start = pos + 1
		default:
			continue
		}
	}

	assert.AssertNotEmpty(hourPart + minutesPart + secondsPart)

	hours, _ := strconv.Atoi(hourPart)
	minutes, _ := strconv.Atoi(minutesPart)
	seconds, _ := strconv.Atoi(secondsPart)

	return time.Duration(time.Hour*time.Duration(hours) + time.Minute*time.Duration(minutes) + time.Second*time.Duration(seconds))
}

func (c *ActivityClient) Activity() Activity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentActivity
}

func (c *ActivityClient) setActivity(a Activity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentActivity = a
	c.lastUpdate.Store(time.Now())
	c.dirty.Store(true)
}

func (c *ActivityClient) ClearActivity() {
	if c.Activity().Id == "Chocola X Vanilla" {
		return
	}

	c.logger.Info("clearing activity")
	c.setActivity(Activity{
		Id:           "Chocola X Vanilla",
		Title:        "(─‿‿─)",
		Author:       "ヾ( ￣O￣)ツ",
		Url:          "https://www.joshuadematas.me/",
		ThumbnailUrl: "https://i.pinimg.com/736x/71/eb/50/71eb502aea2fc4e816b67a5bbd114d27.jpg",
		Duration:     time.Duration(55 * time.Minute),
	})
	c.dirty.Store(true)
}

func (c *ActivityClient) StreamSVG(ctx context.Context, out io.Writer) error {
	if !c.dirty.Load() {
		file, err := os.Open("data/activity.svg")
		if err != nil {
			return err
		}
		_, err = io.Copy(out, file)
		return err
	}

	c.logger.Info("activity dirty, re-building SVG")

	c.fileMu.Lock()
	defer c.fileMu.Unlock()

	file, err := os.OpenFile("data/activity.svg", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	out = io.MultiWriter(file, out)

	templates, err := template.ParseFS(templates, "*/**")
	if err != nil {
		return err
	}

	activity := c.Activity()
	c.logger.Debug("downloading activity thumb", "id", activity.Id, "url", activity.ThumbnailUrl)
	base64Image, err := downloadBase64Image(ctx, activity.ThumbnailUrl)
	if err != nil {
		return fmt.Errorf("image download: %s", err)
	}

	title := fmt.Sprintf("%s - %s", activity.Title, activity.Author)
	input := struct {
		Title        string
		Base64Image  string
		ExternalLink string
	}{
		Title:        html.EscapeString(title),
		Base64Image:  base64Image,
		ExternalLink: html.EscapeString(activity.Url),
	}

	err = templates.ExecuteTemplate(out, ".template.svg", input)
	if err != nil {
		return err
	}

	c.dirty.Store(false)
	return nil
}

func downloadBase64Image(ctx context.Context, url string) (string, error) {
	type result struct {
		img string
		err error
	}
	resch := make(chan result)
	go func() {
		response, err := http.Get(url)
		if err != nil {
			resch <- result{err: err}
			return
		}
		defer response.Body.Close()

		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			resch <- result{err: err}
			return
		}

		str := base64.StdEncoding.EncodeToString(bytes)
		resch <- result{img: fmt.Sprintf("data:image/jpeg;base64,%s", str)}
	}()

	select {
	case res := <-resch:
		return res.img, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
