package prometheus

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/graph"

	"github.com/prometheus/common/model"
)

const (
	// 1: additional filter labels
	queryFormatSuccessRate = `
	sum by (namespace, deployment, statefulset) (
		irate(
			response_total{classification="success", direction="inbound", namespace!="" %[1]s}[120s]
		)
	) /
	sum by (namespace, deployment, statefulset) (
		irate(
			response_total{direction="inbound", namespace!="" %[1]s}[120s]
		)
	) >= 0`

	queryFormatLatencyP95 = `
	histogram_quantile(
		0.95,
		sum by (le, namespace, deployment, statefulset) (
			rate(response_latency_ms_bucket{direction="inbound" %[1]s}[120s])
		)
	)
	`

	// 1: additional filter labels
	queryFormatRequestVolume = `
	sum by (namespace, deployment, statefulset) (
		rate(request_total{direction="inbound" %[1]s}[120s])
	) 
	`

	// 1: additional filter labels
	queryFormatEdges = `
	  sum by (deployment, statefulset, namespace, dst_namespace, dst_deployment, dst_statefulset)
		  (rate(response_total{namespace!="", dst_namespace!="" %[1]s}[120s])
		)
	`
	namespaceLabel      = model.LabelName("namespace")
	dstNamespaceLabel   = model.LabelName("dst_namespace")
	deploymentLabel     = model.LabelName("deployment")
	statefulsetLabel    = model.LabelName("statefulset")
	dstDeploymentLabel  = model.LabelName("dst_deployment")
	dstStatefulsetLabel = model.LabelName("dst_statefulset")
)

type Builder struct {
	client              *Client
	vectorSuccessRate   model.Vector
	vectorLatencyP95    model.Vector
	vectorRequestVolume model.Vector
	vectorEdges         model.Vector
}

func (prometheus Client) NewBuilder() *Builder {
	return &Builder{
		client:              &prometheus,
		vectorSuccessRate:   nil,
		vectorRequestVolume: nil,
		vectorLatencyP95:    nil,
		vectorEdges:         nil,
	}
}

func (builder *Builder) Build(ctx context.Context, from int64, to int64) (*Builder, error) {
	chVectorEdges := make(chan buildVectorResult, 1)
	chVectorSuccessRate := make(chan buildVectorResult, 1)
	chVectorRequestVolume := make(chan buildVectorResult, 1)
	chVectorLatencyP95 := make(chan buildVectorResult, 1)

	go buildVector(ctx,
		from,
		to,
		builder.client,
		chVectorSuccessRate,
		fmt.Sprintf(queryFormatSuccessRate, builder.client.Labels))

	go buildVector(ctx,
		from,
		to,
		builder.client,
		chVectorEdges,
		fmt.Sprintf(
			queryFormatEdges,
			builder.client.Labels))

	go buildVector(ctx,
		from,
		to,
		builder.client,
		chVectorRequestVolume,
		fmt.Sprintf(
			queryFormatRequestVolume,
			builder.client.Labels))

	go buildVector(ctx,
		from,
		to,
		builder.client,
		chVectorLatencyP95,
		fmt.Sprintf(
			queryFormatLatencyP95,
			builder.client.Labels))

	vectorEdges := <-chVectorEdges
	vectorSuccessRate := <-chVectorSuccessRate
	vectorRequestVolume := <-chVectorRequestVolume
	vectorLatencyP95 := <-chVectorLatencyP95

	if vectorEdges.err != nil {
		return nil, fmt.Errorf("failed to build vector edges: %w", vectorEdges.err)
	}

	if vectorSuccessRate.err != nil {
		return nil, fmt.Errorf("failed to build vector success rate: %w", vectorSuccessRate.err)
	}

	if vectorLatencyP95.err != nil {
		return nil, fmt.Errorf("failed to build vector latency : %w", vectorLatencyP95.err)
	}

	if vectorRequestVolume.err != nil {
		return nil, fmt.Errorf("failed to build vector volume: %w", vectorRequestVolume.err)
	}

	builder.vectorEdges = vectorEdges.vector
	builder.vectorSuccessRate = vectorSuccessRate.vector
	builder.vectorLatencyP95 = vectorLatencyP95.vector
	builder.vectorRequestVolume = vectorRequestVolume.vector

	return builder, nil
}

type buildVectorResult struct {
	vector model.Vector
	err    error
}

func buildVector(ctx context.Context, from int64, to int64, client *Client, ch chan buildVectorResult, q string) {
	vector, err := client.queryRange(ctx, q, from, to)
	ch <- buildVectorResult{vector, err}
}

// Node returns the graph.Node associated with resource.
func (builder Builder) Node(ctx context.Context, resource graph.Resource) *graph.Node {
	kind := resourceKindToLabel(resource.Kind)
	namespace := model.LabelValue(resource.Namespace)
	name := model.LabelValue(resource.Name)

	return &graph.Node{
		Resource:      resource,
		SuccessRate:   float64(findInVector(builder.vectorSuccessRate, kind, namespace, name)),
		RequestVolume: float64(findInVector(builder.vectorRequestVolume, kind, namespace, name)),
		LatencyP95:    float64(findInVector(builder.vectorLatencyP95, kind, namespace, name)),
	}
}

func findInVector(vector model.Vector, kind model.LabelName, namespace model.LabelValue, name model.LabelValue) model.SampleValue {
	if vector == nil {
		return 0
	}

	for _, sample := range vector {
		if sample.Metric[namespaceLabel] == namespace && sample.Metric[kind] == name {
			return sample.Value
		}
	}

	return 0
}

func (builder Builder) UpstreamEdgesOf(ctx context.Context, node *graph.Node) []graph.Edge {
	edges := []graph.Edge{}

	if builder.vectorEdges == nil {
		return edges
	}

	for _, sample := range builder.vectorEdges {
		if !validEdgeSample(*sample) {
			continue
		}

		if string(sample.Metric[namespaceLabel]) != node.Resource.Namespace {
			continue
		}

		if v, ok := sample.Metric[deploymentLabel]; ok {
			if string(v) != node.Resource.Name {
				continue
			}
		} else if v, ok := sample.Metric[statefulsetLabel]; ok {
			if string(v) != node.Resource.Name {
				continue
			}
		} else {
			continue
		}

		var resource graph.Resource

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

		edgeNode := builder.Node(ctx, resource)

		edges = append(edges, graph.Edge{Source: node, Destination: edgeNode})
	}

	return edges
}

func (builder Builder) DownstreamEdgesOf(ctx context.Context, node *graph.Node) []graph.Edge {
	edges := []graph.Edge{}

	if builder.vectorEdges == nil {
		return edges
	}

	for _, sample := range builder.vectorEdges {
		if !validEdgeSample(*sample) {
			continue
		}

		if string(sample.Metric[dstNamespaceLabel]) != node.Resource.Namespace {
			continue
		}

		if v, ok := sample.Metric[dstDeploymentLabel]; ok {
			if string(v) != node.Resource.Name {
				continue
			}
		} else if v, ok := sample.Metric[dstStatefulsetLabel]; ok {
			if string(v) != node.Resource.Name {
				continue
			}
		} else {
			continue
		}

		var resource graph.Resource

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

		edgeNode := builder.Node(ctx, resource)

		edges = append(edges, graph.Edge{Source: edgeNode, Destination: node})
	}

	return edges
}

func (builder Builder) EdgesOf(ctx context.Context, node *graph.Node) []graph.Edge {
	edges := []graph.Edge{}
	edges = append(edges, builder.UpstreamEdgesOf(ctx, node)...)
	edges = append(edges, builder.DownstreamEdgesOf(ctx, node)...)

	return edges
}
