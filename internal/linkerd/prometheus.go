package linkerd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var ErrNotAVector = errors.New("expected vector")

type promAPI interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...prom.Option) (model.Value, prom.Warnings, error)
}

type PromGraphSource struct {
	Client promAPI
}

func NewPromGraphSource(address string) (*PromGraphSource, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus graphsource: %w", err)
	}

	return &PromGraphSource{
		Client: prom.NewAPI(client),
	}, nil
}

func (p PromGraphSource) query(ctx context.Context, q string) (model.Vector, error) {
	res, warn, err := p.Client.Query(ctx, q, time.Now())
	if err != nil {
		return nil, fmt.Errorf("query failed: %q: %w", q, err)
	}

	if warn != nil {
		log.Printf("%v", warn)
	}

	if vector, ok := res.(model.Vector); ok {
		return vector, nil
	}

	return nil, fmt.Errorf("received '%s': %w", res.Type(), ErrNotAVector)
}

func (p PromGraphSource) Nodes(ctx context.Context) (*[]Node, error) {
	nodes := []Node{}
	queryFormat := `sum(irate(response_total{classification="success", direction="inbound", %[1]s!="", namespace!=""}[5m])) by (namespace, %[1]s) / sum(irate(response_total{direction="inbound", %[1]s!="", namespace!=""}[5m])) by (namespace, %[1]s) >= 0`

	for _, resourceType := range resourceTypes {
		vector, err := p.query(ctx, fmt.Sprintf(queryFormat, resourceType.String()))
		if err != nil {
			return nil, err
		}

		for _, sample := range vector {
			value := float64(sample.Value)

			nodes = append(nodes, Node{
				SuccessRate: &value,
				Resource: Resource{
					Namespace:    sample.Metric[namespaceLabel],
					ResourceType: resourceType,
					Name:         sample.Metric[resourceType.Label()],
				},
			})
		}
	}

	return &nodes, nil
}

func (p PromGraphSource) Edges(ctx context.Context) (*[]Edge, error) {
	e := []Edge{}

	vector, err := p.query(ctx, "sum(rate(response_total[5m])) by (deployment, statefulset, namespace, dst_namespace, dst_deployment, dst_statefulset)")
	if err != nil {
		return nil, err
	}

	for _, v := range vector {
		edge := parseSample(v)
		// some series are missing fields
		if edge != nil {
			e = append(e, *edge)
		}
	}

	return &e, nil
}

func parseSample(sample *model.Sample) *Edge {
	edge := Edge{}

	if v, ok := sample.Metric[deploymentLabel]; ok {
		edge.Source.ResourceType = DeploymentResourceType
		edge.Source.Name = v
	} else if v, ok := sample.Metric[statefulsetLabel]; ok {
		edge.Source.ResourceType = StatefulsetResourceType
		edge.Source.Name = v
	} else {
		return nil
	}

	if v, ok := sample.Metric[dstDeploymentLabel]; ok {
		edge.Destination.ResourceType = DeploymentResourceType
		edge.Destination.Name = v
	} else if v, ok := sample.Metric[dstStatefulsetLabel]; ok {
		edge.Destination.ResourceType = StatefulsetResourceType
		edge.Destination.Name = v
	} else {
		return nil
	}

	if _, ok := sample.Metric[namespaceLabel]; !ok {
		return nil
	}

	if _, ok := sample.Metric[dstNamespaceLabel]; !ok {
		return nil
	}

	edge.Source.Namespace = sample.Metric[namespaceLabel]
	edge.Destination.Namespace = sample.Metric[dstNamespaceLabel]

	return &edge
}
