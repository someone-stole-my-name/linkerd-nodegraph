package linkerd

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/graph"
	"linkerd-nodegraph/internal/graph/source/prometheus"
	"linkerd-nodegraph/internal/nodegraph"
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
		{Name: "detail__type", Type: nodegraph.FieldTypeString, DisplayName: "Type"},
		{Name: "detail__namespace", Type: nodegraph.FieldTypeString, DisplayName: "Namespace"},
		{Name: "detail__name", Type: nodegraph.FieldTypeString, DisplayName: "Name"},
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

	root, err := m.Server.Node(resource, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain root node: %w", err)
	}

	err = nodeGraph.AddNode(nodegraphNode(*root))
	if err != nil {
		return nil, fmt.Errorf("failed to add root node to graph: %w", err)
	}

	seenNodes := map[string]bool{}
	seenEdges := map[string]bool{}
	currentDepth := 0

	seenNodes[root.Id()] = true
	nodesToScan := []*graph.Node{root}

	for currentDepth < targetDepth {
		currentDepth++
		newNodesToScan := []*graph.Node{}

		for _, node := range nodesToScan {
			edges := []graph.Edge{}
			switch parameters.Direction {
			case "inbound":
				edges, err = m.Server.DownstreamEdgesOf(node, ctx)
			case "outbound":
				edges, err = m.Server.UpstreamEdgesOf(node, ctx)
			default:
				edges, err = m.Server.EdgesOf(node, ctx)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to obtain the list of edges: %w", err)
			}

			for _, edge := range edges {
				if ok, _ := seenNodes[edge.Source.Id()]; !ok {
					seenNodes[edge.Source.Id()] = true
					err = nodeGraph.AddNode(nodegraphNode(*edge.Source))
					newNodesToScan = append(nodesToScan, edge.Source)
					if err != nil {
						return nil, fmt.Errorf("failed to add node: %w", err)
					}
				}

				if ok, _ := seenNodes[edge.Destination.Id()]; !ok {
					seenNodes[edge.Destination.Id()] = true
					err = nodeGraph.AddNode(nodegraphNode(*edge.Destination))
					newNodesToScan = append(nodesToScan, edge.Destination)
					if err != nil {
						return nil, fmt.Errorf("failed to add node: %w", err)
					}
				}

				if ok, _ := seenEdges[edge.Id()]; !ok {
					seenEdges[edge.Id()] = true
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
		Kind:      graph.KindFromString(p.Kind),
	}

	return resource
}

func nodegraphEdge(edge graph.Edge) nodegraph.Edge {
	return nodegraph.Edge{
		"id":     edge.Id(),
		"source": edge.Source.Id(),
		"target": edge.Destination.Id(),
	}
}

func nodegraphNode(node graph.Node) nodegraph.Node {
	var success float64 = 0
	var failed float64 = 1
	percent := "N/A"

	if node.SuccessRate != nil {
		success = *node.SuccessRate
		failed = 1 - success
		percent = fmt.Sprintf("%.2f%%", success*100) //nolint:gomnd
	}

	return nodegraph.Node{
		"id":                node.Id(),
		"title":             fmt.Sprintf("%s/%s", node.Resource.Namespace, node.Resource.Name),
		"arc__failed":       failed,
		"arc__success":      success,
		"detail__type":      node.Resource.Kind.String(),
		"detail__namespace": node.Resource.Namespace,
		"detail__name":      node.Resource.Name,
		"mainStat":          percent,
	}
}
