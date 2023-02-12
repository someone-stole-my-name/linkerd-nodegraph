package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type promAPI interface {
	QueryRange(ctx context.Context, query string, r prom.Range, opts ...prom.Option) (model.Value, prom.Warnings, error)
}

type Client struct {
	API    promAPI
	Labels string
}

type roundTripper struct {
	headers   map[string]string
	transport http.RoundTripper
}

type Config struct {
	Address   string
	Labels    string
	Headers   map[string]string
	TLSConfig *tls.Config
}

func NewClient(config Config) (*Client, error) {
	c, err := api.NewClient(api.Config{
		Address: config.Address,
		Client: &http.Client{
			Transport: &roundTripper{
				headers: config.Headers,
				transport: &http.Transport{
					TLSClientConfig: config.TLSConfig,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	if config.Labels != "" && config.Labels != " " {
		config.Labels = "," + config.Labels
	} else {
		config.Labels = " "
	}

	return &Client{
		API:    prom.NewAPI(c),
		Labels: config.Labels,
	}, nil
}

func (t *roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		r.Header.Set(k, v)
	}

	return t.transport.RoundTrip(r)
}
