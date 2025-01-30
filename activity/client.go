package activity

import (
	"context"
	"fmt"
	"io"
	"sync"
	"text/template"

	"github.com/charmbracelet/log"

	"github.com/tifye/shigure/assert"
)

type Activity struct {
	Id           string
	Title        string
	Author       string
	Url          string
	ThumbnailUrl string
}

type Client struct {
	apiKey string
	logger *log.Logger

	currentActivity Activity
	dirty           bool
	mu              sync.RWMutex
}

func NewClient(logger *log.Logger, apiKey string) *Client {
	assert.AssertNotNil(logger)
	assert.AssertNotEmpty(apiKey)
	return &Client{
		logger: logger,
		apiKey: apiKey,
		currentActivity: Activity{
			Id:           "Chocola X Vanilla",
			Title:        "(─‿‿─)",
			Author:       "ヾ( ￣O￣)ツ",
			Url:          "https://www.joshuadematas.me/",
			ThumbnailUrl: "https://i.pinimg.com/736x/71/eb/50/71eb502aea2fc4e816b67a5bbd114d27.jpg",
		},
		mu: sync.RWMutex{},
	}
}

func (c *Client) SetYoutubeActivity(ctx context.Context, videoId string) error {
	assert.AssertNotEmpty(videoId)

	if c.Activity().Id == videoId {
		return nil
	}

	resource, err := FetchYoutubeVideoResource(ctx, c.logger, c.apiKey, videoId)
	if err != nil {
		return fmt.Errorf("fetch video: %s", err)
	}

	c.setActivity(Activity{
		Id:           resource.Id,
		Url:          fmt.Sprintf("https://www.youtube.com/watch?v=%s", resource.Id),
		Title:        resource.Snippet.Title,
		Author:       resource.Snippet.ChannelTitle,
		ThumbnailUrl: resource.Snippet.Thumbnails.HighRes.Url,
	})

	return nil
}

func (c *Client) Activity() Activity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentActivity
}

func (c *Client) setActivity(a Activity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentActivity = a
	c.dirty = true
}

func (c *Client) StreamSVG(out io.Writer) error {
	templates, err := template.ParseFiles("./activity/templates/.template.svg")
	if err != nil {
		return err
	}

	activity := c.Activity()

	input := struct {
		Title        string
		ImageSource  string
		ExternalLink string
	}{
		Title:        fmt.Sprintf("%s - %s", activity.Title, activity.Author),
		ImageSource:  activity.ThumbnailUrl,
		ExternalLink: activity.Url,
	}

	return templates.ExecuteTemplate(out, ".template.svg", input)
}
