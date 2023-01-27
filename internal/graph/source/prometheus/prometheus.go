package prometheus

import (
	"context"
	"errors"
	"fmt"
	"linkerd-nodegraph/internal/graph"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	namespaceLabel      = model.LabelName("namespace")
	dstNamespaceLabel   = model.LabelName("dst_namespace")
	deploymentLabel     = model.LabelName("deployment")
	statefulsetLabel    = model.LabelName("statefulset")
	dstDeploymentLabel  = model.LabelName("dst_deployment")
	dstStatefulsetLabel = model.LabelName("dst_statefulset")
)

var ErrNotAVector = errors.New("expected vector")

type promAPI interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...prom.Option) (model.Value, prom.Warnings, error)
}

type Client struct {
	API promAPI
}

func NewClient(address string) (*Client, error) {
	c, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	return &Client{
		API: prom.NewAPI(c),
	}, nil
}

func (prometheus Client) query(ctx context.Context, q string) (model.Vector, error) {
	res, warn, err := prometheus.API.Query(ctx, q, time.Now())
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

// Node returns the graph.Node associated with resource
func (prometheus Client) Node(resource graph.Resource, ctx context.Context) (*graph.Node, error) {
	var node graph.Node
	node.Name = resource.Name
	node.Namespace = resource.Namespace
	node.Type = resource.Type

	vectorSuccessRateSingle, err := prometheus.query(ctx, fmt.Sprintf(
		queryFormatSuccessRateSingle,
		resource.Type.String(),
		resource.Name,
		resource.Namespace),
	)
	if err != nil {
		return nil, err
	}

	if len(vectorSuccessRateSingle) != 1 {
		return nil, fmt.Errorf("expected a vector with a single sample, got %d", len(vectorSuccessRateSingle))
	}

	successRate := float64(vectorSuccessRateSingle[0].Value)
	node.SuccessRate = &successRate

	return &node, nil
}

func (prometheus Client) EdgesOf(node *graph.Node, ctx context.Context) ([]graph.Edge, error) {
	edges := []graph.Edge{}

	vectorEdgesOfUpstreams, err := prometheus.query(ctx, fmt.Sprintf(
		queryFormatEdgesOfUpstreams,
		node.Type.String(),
		node.Name,
		node.Namespace),
	)
	if err != nil {
		return edges, err
	}

	vectorEdgesOfDownstreams, err := prometheus.query(ctx, fmt.Sprintf(
		queryFormatEdgesOfDownstreams,
		node.Type.String(),
		node.Name,
		node.Namespace),
	)
	if err != nil {
		return edges, err
	}

	for _, sample := range vectorEdgesOfUpstreams {
		var resource graph.Resource

		if _, ok := sample.Metric[dstNamespaceLabel]; !ok {
			continue
		}

		resource.Namespace = string(sample.Metric[dstNamespaceLabel])

		if v, ok := sample.Metric[dstDeploymentLabel]; ok {
			resource.Type = graph.DeploymentResourceType
			resource.Name = string(v)
		} else if v, ok := sample.Metric[dstStatefulsetLabel]; ok {
			resource.Type = graph.StatefulsetResourceType
			resource.Name = string(v)
		} else {
			continue
		}

		edgeNode, err := prometheus.Node(resource, ctx)
		if err != nil {
			continue
		}

		edges = append(edges, graph.Edge{Source: node, Destination: edgeNode})
	}

	for _, sample := range vectorEdgesOfDownstreams {
		var resource graph.Resource

		if _, ok := sample.Metric[namespaceLabel]; !ok {
			continue
		}

		resource.Namespace = string(sample.Metric[namespaceLabel])

		if v, ok := sample.Metric[deploymentLabel]; ok {
			resource.Type = graph.DeploymentResourceType
			resource.Name = string(v)
		} else if v, ok := sample.Metric[statefulsetLabel]; ok {
			resource.Type = graph.StatefulsetResourceType
			resource.Name = string(v)
		} else {
			continue
		}

		edgeNode, err := prometheus.Node(resource, ctx)
		if err != nil {
			continue
		}

		edges = append(edges, graph.Edge{Source: edgeNode, Destination: node})
	}

	return edges, nil
}
