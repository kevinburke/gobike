// Package client is a client for retrieving data from the GoBike Server.
package client

import (
	"io"
	"net/http"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/rest"
)

// Metadata about response
type Response struct {
	LastUpdated time.Time
	TTL         int
}

type response struct {
	LastUpdated int64 `json:"last_updated"`
	TTL         int   `json:"ttl"`
}

type Client struct {
	Client *rest.Client
	Host   string

	Stations *StationService
}

const Host = "https://gbfs.fordgobike.com/gbfs/en"

// NewClient returns a new Client.
func NewClient() *Client {
	c := new(Client)
	c.Host = Host
	c.Client = rest.NewClient("", "", Host)

	c.Stations = &StationService{c}
	return c
}

// NewRequest creates a new HTTP request to hit the given endpoint.
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := c.Client.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gobike/"+gobike.Version+" (github.com/kevinburke/gobike) "+req.Header.Get("User-Agent"))
	return req, nil
}
