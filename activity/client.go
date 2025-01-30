package activity

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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

func (c *Client) StreamSVG(ctx context.Context, out io.Writer) error {
	templates, err := template.ParseFiles("./activity/templates/.template.svg")
	if err != nil {
		return err
	}

	activity := c.Activity()
	base64Image, err := downloadBase64Image(ctx, activity.ThumbnailUrl)
	if err != nil {
		return fmt.Errorf("image download: %s", err)
	}

	input := struct {
		Title        string
		Base64Image  string
		ExternalLink string
	}{
		Title:        fmt.Sprintf("%s - %s", activity.Title, activity.Author),
		Base64Image:  base64Image,
		ExternalLink: activity.Url,
	}

	return templates.ExecuteTemplate(out, ".template.svg", input)
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
