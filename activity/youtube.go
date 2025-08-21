package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/charmbracelet/log"
)

type YoutubeVideoListResponse struct {
	Items []YoutubeVideoResource `json:"items"`
}

type YoutubeVideoResource struct {
	Id      string `json:"id"`
	Snippet struct {
		Title        string `json:"title"`
		ChannelTitle string `json:"channelTitle"`
		Thumbnails   struct {
			HighRes   YoutubeThumbnailData `json:"high"`
			MediumRes YoutubeThumbnailData `json:"medium"`
		} `json:"thumbnails"`
	} `json:"snippet"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
}

type YoutubeThumbnailData struct {
	Url    string `json:"url"`
	Width  uint   `json:"width"`
	Height uint   `json:"height"`
}

func FetchYoutubeVideoResource(
	ctx context.Context,
	logger *log.Logger,
	apiKey string,
	videoId string,
) (YoutubeVideoResource, error) {
	url, err := url.Parse("https://youtube.googleapis.com/youtube/v3/videos")
	if err != nil {
		return YoutubeVideoResource{}, err
	}
	query := url.Query()
	query.Add("part", "snippet,contentDetails")
	query.Add("id", videoId)
	query.Add("key", apiKey)
	url.RawQuery = query.Encode()
	res, err := http.Get(url.String())
	if err != nil {
		return YoutubeVideoResource{}, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return YoutubeVideoResource{}, err
	}
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logger.Error("body close: %s", err)
		}
	}()

	if res.StatusCode > 299 {
		return YoutubeVideoResource{}, fmt.Errorf("response failed with code %d\nand body: %s", res.StatusCode, body)
	}

	var resp YoutubeVideoListResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return YoutubeVideoResource{}, err
	}

	if len(resp.Items) <= 0 {
		return YoutubeVideoResource{}, fmt.Errorf("could not find video resource for %s", videoId)
	}

	return resp.Items[0], nil
}
