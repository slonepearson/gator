package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func (r *RSSFeed) UnescapeRssFeed() {
	r.Channel.Title = html.UnescapeString(r.Channel.Title)
	r.Channel.Description = html.UnescapeString(r.Channel.Description)
	for i, item := range r.Channel.Item {
		r.Channel.Item[i].Title = html.UnescapeString(item.Title)
		r.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}
}

func FetchFeed(ctx context.Context, feedUrl string) (*RSSFeed, error) {
	if feedUrl == "" {
		return &RSSFeed{}, fmt.Errorf("expected feedUrl got: %s", feedUrl)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", feedUrl, nil)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("problem creating request: %w", err)
	}
	req.Header.Set("User-Agent", "gator")
	client := http.Client{Timeout: time.Second * 30}

	res, err := client.Do(req)

	if err != nil {
		return &RSSFeed{}, fmt.Errorf("problem with request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode > 300 || res.StatusCode < 200 {
		return &RSSFeed{}, fmt.Errorf("non ok GET request: %v", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("problem reading response: %w", err)
	}
	var rssFeed RSSFeed

	if err := xml.Unmarshal(data, &rssFeed); err != nil {
		return &RSSFeed{}, fmt.Errorf("problem parsing xml: %w", err)
	}

	return &rssFeed, nil
}
