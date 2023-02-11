package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	headers map[string]string
}

func NewClient(address string, labels string, headers string) (*Client, error) {
	headersMap := make(map[string]string)

	if len(headers) != 0 {
		for _, header := range strings.Split(headers, ",") {
			kv := strings.Split(header, "=")
			if len(kv) != 2 {
				return nil, fmt.Errorf("expected key value found: %v", kv)
			}

			headersMap[kv[0]] = kv[1]
		}
	}

	c, err := api.NewClient(api.Config{
		Address: address,
		RoundTripper: &roundTripper{
			headers: headersMap,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	if labels != "" && labels != " " {
		labels = "," + labels
	} else {
		labels = " "
	}

	return &Client{
		API:    prom.NewAPI(c),
		Labels: labels,
	}, nil
}

func (t *roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		r.Header.Set(k, v)
	}

	return http.DefaultTransport.RoundTrip(r)
}
