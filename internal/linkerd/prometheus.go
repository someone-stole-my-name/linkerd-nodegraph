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
		return nil, err
	}

	return &PromGraphSource{
		Client: prom.NewAPI(client),
	}, nil
}

func (p PromGraphSource) query(ctx context.Context, q string) (model.Vector, error) {
	res, warn, err := p.Client.Query(ctx, q, time.Now())
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

func (p PromGraphSource) Nodes(ctx context.Context) (*[]Node, error) {
	nodes := []Node{}
	queryFormat := `sum(irate(response_total{classification="success", direction="inbound", %[1]s!="", namespace!=""}[5m])) by (namespace, %[1]s) / sum(irate(response_total{direction="inbound", %[1]s!="", namespace!=""}[5m])) by (namespace, %[1]s) >= 0`

	for _, resourceType := range resourceTypes {
		vector, err := p.query(ctx, fmt.Sprintf(queryFormat, resourceType.String()))
		if err != nil {
			return nil, err
		}

		for _, v := range vector {
			value := float64(v.Value)

			nodes = append(nodes, Node{
				SuccessRate: &value,
				Resource: Resource{
					Namespace:    v.Metric[namespaceLabel],
					ResourceType: resourceType,
					Name:         v.Metric[resourceType.Label()],
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

func parseSample(s *model.Sample) *Edge {
	e := Edge{}

	if v, ok := s.Metric[deploymentLabel]; ok {
		e.Source.ResourceType = DeploymentResourceType
		e.Source.Name = v
	} else if v, ok := s.Metric[statefulsetLabel]; ok {
		e.Source.ResourceType = StatefulsetResourceType
		e.Source.Name = v
	} else {
		return nil
	}

	if v, ok := s.Metric[dstDeploymentLabel]; ok {
		e.Destination.ResourceType = DeploymentResourceType
		e.Destination.Name = v
	} else if v, ok := s.Metric[dstStatefulsetLabel]; ok {
		e.Destination.ResourceType = StatefulsetResourceType
		e.Destination.Name = v
	} else {
		return nil
	}

	if _, ok := s.Metric[namespaceLabel]; !ok {
		return nil
	}

	if _, ok := s.Metric[dstNamespaceLabel]; !ok {
		return nil
	}

	e.Source.Namespace = s.Metric[namespaceLabel]
	e.Destination.Namespace = s.Metric[dstNamespaceLabel]

	return &e
}
