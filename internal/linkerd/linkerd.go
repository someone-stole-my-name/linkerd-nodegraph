package linkerd

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/graph"
	"linkerd-nodegraph/internal/graph/source/prometheus"
	"linkerd-nodegraph/internal/nodegraph"
)

const (
	defaultUnknownValue = "N/A"
)

type Stats struct {
	Server *prometheus.Client
}

type Parameters struct {
	Depth     int    `schema:"depth"`
	Name      string `schema:"name"`
	Namespace string `schema:"namespace"`
	Kind      string `schema:"kind"`
	Direction string `schema:"direction"`
	From      int64  `schema:"from"`
	To        int64  `schema:"to"`
}

var GraphSpec = nodegraph.NodeFields{
	Edge: []nodegraph.Field{
		{Name: "id", Type: nodegraph.FieldTypeString},
		{Name: "source", Type: nodegraph.FieldTypeString},
		{Name: "target", Type: nodegraph.FieldTypeString},
	},
	Node: []nodegraph.Field{
		{Name: "id", Type: nodegraph.FieldTypeString},
		{Name: "title", Type: nodegraph.FieldTypeString, DisplayName: "Resource"},
		{Name: "mainStat", Type: nodegraph.FieldTypeString, DisplayName: "Success Rate"},
		{Name: "secondaryStat", Type: nodegraph.FieldTypeString, DisplayName: "Latency"},
		{Name: "detail__type", Type: nodegraph.FieldTypeString, DisplayName: "Type"},
		{Name: "detail__namespace", Type: nodegraph.FieldTypeString, DisplayName: "Namespace"},
		{Name: "detail__name", Type: nodegraph.FieldTypeString, DisplayName: "Name"},
		{Name: "detail__successRate", Type: nodegraph.FieldTypeString, DisplayName: "Success Rate"},
		{Name: "detail__latency_p95", Type: nodegraph.FieldTypeString, DisplayName: "p95"},
		{Name: "detail__volume", Type: nodegraph.FieldTypeString, DisplayName: "Request volume"},
		{
			Name:        "arc__failed",
			Type:        nodegraph.FieldTypeNumber,
			Color:       "red",
			DisplayName: "Failed",
		},
		{
			Name:        "arc__success",
			Type:        nodegraph.FieldTypeNumber,
			Color:       "green",
			DisplayName: "Success",
		},
	},
}

func (m Stats) Graph(ctx context.Context, parameters Parameters) (*nodegraph.Graph, error) {
	nodeGraph := nodegraph.Graph{
		Spec:  GraphSpec,
		Nodes: []nodegraph.Node{},
		Edges: []nodegraph.Edge{},
	}

	resource := parameters.graphResource()

	targetDepth := 1
	if parameters.Depth != 0 {
		targetDepth = parameters.Depth
	}

	b, err := m.Server.NewBuilder().Build(ctx, parameters.From, parameters.To)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	root := b.Node(ctx, resource)

	err = nodeGraph.AddNode(nodegraphNode(*root))
	if err != nil {
		return nil, fmt.Errorf("failed to add root node to graph: %w", err)
	}

	seenNodes := map[string]bool{}
	seenEdges := map[string]bool{}
	currentDepth := 0

	seenNodes[root.ID()] = true
	nodesToScan := []*graph.Node{root}

	var edgesFunc func(context.Context, *graph.Node) []graph.Edge

	switch parameters.Direction {
	case "inbound":
		edgesFunc = b.DownstreamEdgesOf
	case "outbound":
		edgesFunc = b.UpstreamEdgesOf
	default:
		edgesFunc = b.EdgesOf
	}

	for currentDepth < targetDepth {
		currentDepth++

		newNodesToScan := []*graph.Node{}

		for _, node := range nodesToScan {
			edges := edgesFunc(ctx, node)

			for _, edge := range edges {
				if ok := seenNodes[edge.Source.ID()]; !ok {
					newNodesToScan = append(newNodesToScan, edge.Source)
					seenNodes[edge.Source.ID()] = true

					err = nodeGraph.AddNode(nodegraphNode(*edge.Source))
					if err != nil {
						return nil, fmt.Errorf("failed to add node: %w", err)
					}
				}

				if ok := seenNodes[edge.Destination.ID()]; !ok {
					newNodesToScan = append(newNodesToScan, edge.Destination)
					seenNodes[edge.Destination.ID()] = true

					err = nodeGraph.AddNode(nodegraphNode(*edge.Destination))
					if err != nil {
						return nil, fmt.Errorf("failed to add node: %w", err)
					}
				}

				if ok := seenEdges[edge.ID()]; !ok {
					seenEdges[edge.ID()] = true

					err = nodeGraph.AddEdge(nodegraphEdge(edge))
					if err != nil {
						return nil, fmt.Errorf("failed to add edge: %w", err)
					}
				}
			}
		}

		nodesToScan = newNodesToScan
	}

	return &nodeGraph, nil
}

func (p Parameters) graphResource() graph.Resource {
	resource := graph.Resource{
		Name:      p.Name,
		Namespace: p.Namespace,
		Kind:      graph.ResourceKindFromString(p.Kind),
	}

	return resource
}

func nodegraphEdge(edge graph.Edge) nodegraph.Edge {
	return nodegraph.Edge{
		"id":     edge.ID(),
		"source": edge.Source.ID(),
		"target": edge.Destination.ID(),
	}
}

func nodegraphNode(node graph.Node) nodegraph.Node {
	var failed float64 = 1

	var success float64

	percent := defaultUnknownValue
	p95 := defaultUnknownValue
	volume := defaultUnknownValue

	if node.SuccessRate != 0 {
		success = node.SuccessRate
		failed = 1 - success
		percent = fmt.Sprintf("%.2f%%", success*100) //nolint:gomnd
	}

	if node.LatencyP95 != 0 {
		p95 = fmt.Sprintf("%.1fms", node.LatencyP95)
	}

	if node.RequestVolume != 0 {
		volume = fmt.Sprintf("%.0frd/s", node.RequestVolume)
	}

	return nodegraph.Node{
		"id":                  node.ID(),
		"title":               fmt.Sprintf("%s/%s", node.Resource.Namespace, node.Resource.Name),
		"arc__failed":         failed,
		"arc__success":        success,
		"detail__type":        node.Resource.Kind.String(),
		"detail__namespace":   node.Resource.Namespace,
		"detail__name":        node.Resource.Name,
		"detail__successRate": percent,
		"detail__latency_p95": p95,
		"detail__volume":      volume,
		"mainStat":            "SR: " + percent,
		"secondaryStat":       "p95: " + p95,
	}
}
