package rss

import (
	"context"
	"database/sql"
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

func NewClient() *HttpClient {
	return &HttpClient{
		Client: &http.Client{},
	}
}

func (c *HttpClient) get(ctx context.Context, modifiedSince sql.NullTime, url string) (io.ReadCloser, string, error) {
	if url == "" {
		return nil, "", fmt.Errorf("expected feedUrl got: %s", url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("problem creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	if modifiedSince.Valid {
		req.Header.Set("Last-Modified", modifiedSince.Time.Format(time.RFC1123))
	}

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("problem with request: %w", err)
	}
	if res.StatusCode < 400 && res.StatusCode > 299 {
		return nil, "", fmt.Errorf("Redirected: %v", res.Status)
	}
	if res.StatusCode > 299 || res.StatusCode < 200 {
		res.Body.Close()
		return nil, "", fmt.Errorf("non ok GET request: %v", res.Status)
	}

	lastModified := res.Header.Get("Last-Modified")

	return res.Body, lastModified, nil
}

func (r *RSSFeed) UnescapeRssFeed() {
	r.Channel.Title = html.UnescapeString(r.Channel.Title)
	r.Channel.Description = html.UnescapeString(r.Channel.Description)
	for i, item := range r.Channel.Item {
		r.Channel.Item[i].Title = html.UnescapeString(item.Title)
		r.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}
}

func FetchFeed(ctx context.Context, client *HttpClient, modifiedSince sql.NullTime, url string) (*RSSFeed, string, error) {
	body, lastModified, err := client.get(ctx, modifiedSince, url)
	if err != nil {
		return nil, "", err
	}
	defer body.Close()
	var data RSSFeed

	decoder := xml.NewDecoder(body)

	if err := decoder.Decode(&data); err != nil {
		return nil, "", err
	}

	return &data, lastModified, nil
}
