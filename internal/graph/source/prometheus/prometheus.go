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
	API    promAPI
	Labels string
}

func NewClient(address string, labels string) (*Client, error) {
	c, err := api.NewClient(api.Config{
		Address: address,
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

// Same as query but returns a single sample instead.
func (prometheus Client) querySingle(ctx context.Context, q string) (*model.Sample, error) {
	vector, err := prometheus.query(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(vector) != 1 {
		return nil, fmt.Errorf("expected a vector with a single sample, got %d", len(vector))
	}

	return vector[0], nil
}

// Node returns the graph.Node associated with resource.
func (prometheus Client) Node(ctx context.Context, resource graph.Resource) (*graph.Node, error) {
	var node graph.Node
	node.Resource = resource

	anyTypeNameNamespace := []any{
		resource.Kind.String(),
		resource.Name,
		resource.Namespace,
		prometheus.Labels,
	}

	vectorSuccessRateSingle, err := prometheus.querySingle(ctx, fmt.Sprintf(
		queryFormatSuccessRateSingle,
		anyTypeNameNamespace...))
	if err != nil {
		return nil, err
	}

	vectorLatencyP95Single, err := prometheus.querySingle(ctx, fmt.Sprintf(
		queryFormatLatencyP95Single,
		anyTypeNameNamespace...))
	if err != nil {
		return nil, err
	}

	vectorRequestVolumeSingle, err := prometheus.querySingle(ctx, fmt.Sprintf(
		queryFormatRequestVolumeSingle,
		anyTypeNameNamespace...))
	if err != nil {
		return nil, err
	}

	node.SuccessRate = float64(vectorSuccessRateSingle.Value)
	node.LatencyP95 = float64(vectorLatencyP95Single.Value)
	node.RequestVolume = float64(vectorRequestVolumeSingle.Value)

	return &node, nil
}

func (prometheus Client) DownstreamEdgesOf(ctx context.Context, node *graph.Node) ([]graph.Edge, error) {
	edges := []graph.Edge{}

	anyTypeNameNamespace := []any{
		node.Resource.Kind.String(),
		node.Resource.Name,
		node.Resource.Namespace,
		prometheus.Labels,
	}

	vectorEdgesOfDownstreams, err := prometheus.query(ctx, fmt.Sprintf(
		queryFormatEdgesOfDownstreams,
		anyTypeNameNamespace...),
	)
	if err != nil {
		return edges, err
	}

	for _, sample := range vectorEdgesOfDownstreams {
		var resource graph.Resource

		if _, ok := sample.Metric[namespaceLabel]; !ok {
			continue
		}

		resource.Namespace = string(sample.Metric[namespaceLabel])

		if v, ok := sample.Metric[deploymentLabel]; ok {
			resource.Kind = graph.DeploymentKind
			resource.Name = string(v)
		} else if v, ok := sample.Metric[statefulsetLabel]; ok {
			resource.Kind = graph.StatefulsetKind
			resource.Name = string(v)
		} else {
			continue
		}

		edgeNode, err := prometheus.Node(ctx, resource)
		if err != nil {
			continue
		}

		edges = append(edges, graph.Edge{Source: edgeNode, Destination: node})
	}

	return edges, nil
}

func (prometheus Client) UpstreamEdgesOf(ctx context.Context, node *graph.Node) ([]graph.Edge, error) {
	edges := []graph.Edge{}

	anyTypeNameNamespace := []any{
		node.Resource.Kind.String(),
		node.Resource.Name,
		node.Resource.Namespace,
		prometheus.Labels,
	}

	vectorEdgesOfUpstreams, err := prometheus.query(ctx, fmt.Sprintf(
		queryFormatEdgesOfUpstreams,
		anyTypeNameNamespace...),
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
			resource.Kind = graph.DeploymentKind
			resource.Name = string(v)
		} else if v, ok := sample.Metric[dstStatefulsetLabel]; ok {
			resource.Kind = graph.StatefulsetKind
			resource.Name = string(v)
		} else {
			continue
		}

		edgeNode, err := prometheus.Node(ctx, resource)
		if err != nil {
			continue
		}

		edges = append(edges, graph.Edge{Source: node, Destination: edgeNode})
	}

	return edges, nil
}

func (prometheus Client) EdgesOf(ctx context.Context, node *graph.Node) ([]graph.Edge, error) {
	edges := []graph.Edge{}

	upstreams, err := prometheus.UpstreamEdgesOf(ctx, node)
	if err != nil {
		return edges, err
	}

	edges = append(edges, upstreams...)

	downstreams, err := prometheus.DownstreamEdgesOf(ctx, node)
	if err != nil {
		return edges, err
	}

	edges = append(edges, downstreams...)

	return edges, nil
}
