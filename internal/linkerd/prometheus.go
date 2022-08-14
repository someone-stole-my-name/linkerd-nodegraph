package linkerd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type promGraphSource struct {
	client prom.API
}

func NewPromGraphSource(address string) (*promGraphSource, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, err
	}
	return &promGraphSource{
		client: prom.NewAPI(client),
	}, nil
}

func (p promGraphSource) Query(ctx context.Context, q string) (model.Vector, error) {
	res, warn, err := p.client.Query(ctx, q, time.Now())
	if err != nil {
		return nil, fmt.Errorf("Query failed: %q: %w", q, err)
	}
	if warn != nil {
		log.Printf("%v", warn)
	}
	if res.Type() != model.ValVector {
		return nil, fmt.Errorf("Expected Vector but got: %s", res.Type())
	}
	return res.(model.Vector), nil
}
