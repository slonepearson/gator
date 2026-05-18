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

const userAgent = "gator"

type HttpClient struct {
	Client *http.Client
}

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

func NewClient(t time.Duration) *HttpClient {
	return &HttpClient{
		Client: &http.Client{Timeout: t},
	}
}

func (c *HttpClient) get(ctx context.Context, url string) (io.ReadCloser, error) {
	if url == "" {
		return nil, fmt.Errorf("expected feedUrl got: %s", url)
	}
	ctx, cancle := context.WithTimeout(ctx, 2*time.Second)
	defer cancle()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("problem creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := c.Client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("problem with request: %w", err)
	}

	if res.StatusCode > 299 || res.StatusCode < 200 {
		res.Body.Close()
		return nil, fmt.Errorf("non ok GET request: %v", res.Status)
	}

	return res.Body, nil
}

func (r *RSSFeed) UnescapeRssFeed() {
	r.Channel.Title = html.UnescapeString(r.Channel.Title)
	r.Channel.Description = html.UnescapeString(r.Channel.Description)
	for i, item := range r.Channel.Item {
		r.Channel.Item[i].Title = html.UnescapeString(item.Title)
		r.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}
}

func FetchFeed(client *HttpClient, ctx context.Context, url string) (RSSFeed, error) {
	body, err := client.get(ctx, url)
	if err != nil {
		return RSSFeed{}, err
	}
	defer body.Close()
	var data RSSFeed

	decoder := xml.NewDecoder(body)

	if err := decoder.Decode(&data); err != nil {
		return RSSFeed{}, err
	}

	return data, nil
}
