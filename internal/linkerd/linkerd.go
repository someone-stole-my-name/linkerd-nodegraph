package linkerd

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/nodegraph"

	"github.com/prometheus/common/model"
)

type trafficDirection int

const (
	inbound trafficDirection = iota
	outbound
	unknown
)

type graphSource interface {
	Nodes(ctx context.Context) (*[]Node, error)
	Edges(ctx context.Context) (*[]Edge, error)
}

type Stats struct {
	Server graphSource
}

type Parameters struct {
	Depth     int    `schema:"depth"`
	Direction string `schema:"direction"`
	Root      string `schema:"root"`
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

const (
	namespaceLabel      = model.LabelName("namespace")
	dstNamespaceLabel   = model.LabelName("dst_namespace")
	deploymentLabel     = model.LabelName("deployment")
	statefulsetLabel    = model.LabelName("statefulset")
	dstDeploymentLabel  = model.LabelName("dst_deployment")
	dstStatefulsetLabel = model.LabelName("dst_statefulset")
)

func (m Stats) Graph(ctx context.Context, parameters Parameters) (*nodegraph.Graph, error) {
	graph := nodegraph.Graph{
		Spec:  GraphSpec,
		Nodes: []nodegraph.Node{},
		Edges: []nodegraph.Edge{},
	}

	nodes, err := m.Server.Nodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain the list of nodes: %w", err)
	}

	edges, err := m.Server.Edges(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain the list of edges: %w", err)
	}

	addUnmeshed(edges, nodes)

	if parameters.Root != "" {
		switch parameters.Direction {
		case "inbound":
			setRoot(parameters.Root, parameters.Depth, edges, nodes, inbound)
		case "outbound":
			setRoot(parameters.Root, parameters.Depth, edges, nodes, outbound)
		default:
			setRoot(parameters.Root, parameters.Depth, edges, nodes, unknown)
		}
	} else {
		removeOrphans(edges, nodes)
	}

	for _, node := range *nodes {
		err := graph.AddNode(node.nodegraphNode())
		if err != nil {
			return nil, fmt.Errorf("failed to add node to graph: %w", err)
		}
	}

	for _, edge := range *edges {
		err := graph.AddEdge(edge.nodegraphEdge())
		if err != nil {
			return nil, fmt.Errorf("failed to add edge to graph: %w", err)
		}
	}

	return &graph, nil
}

func addUnmeshed(edges *[]Edge, nodes *[]Node) {
	seenIds := map[string]bool{}

	for _, node := range *nodes {
		seenIds[node.Resource.id()] = true
	}

	for _, edge := range *edges {
		for _, resource := range []Resource{edge.Source, edge.Destination} {
			if _, ok := seenIds[resource.id()]; !ok {
				*nodes = append(*nodes, Node{Resource: resource})
			}
		}
	}
}

func removeOrphans(edges *[]Edge, nodes *[]Node) {
	seenIds := map[string]bool{}
	newNodes := []Node{}

	for _, edge := range *edges {
		seenIds[edge.Destination.id()] = true
		seenIds[edge.Source.id()] = true
	}

	for _, node := range *nodes {
		if _, ok := seenIds[node.Resource.id()]; ok {
			newNodes = append(newNodes, node)
		}
	}

	*nodes = newNodes
}

func removeID(id string, edges *[]Edge, nodes *[]Node) {
	newNodes := []Node{}
	newEdges := []Edge{}

	for _, node := range *nodes {
		if node.Resource.id() != id {
			newNodes = append(newNodes, node)
		}
	}

	for _, edge := range *edges {
		if edge.Source.id() != id && edge.Destination.id() != id {
			newEdges = append(newEdges, edge)
		}
	}

	*nodes = newNodes
	*edges = newEdges
}

func setRoot(id string, depth int, edges *[]Edge, nodes *[]Node, direction trafficDirection) {
	rootExists := false

	for _, node := range *nodes {
		if node.Resource.id() == id {
			rootExists = true

			break
		}
	}

	if !rootExists {
		return
	}

	currentDepth := 0
	connectedNodeIds := map[string]bool{}
	connectedNodeIds[id] = true

	for currentDepth != depth {
		iterationNodeIds := map[string]bool{}
		currentDepth++

		for root := range connectedNodeIds {
			var ids []string

			switch direction {
			case inbound:
				ids = findInboundNodesConnectedTo(root, *edges)
			case outbound:
				ids = findOutboundNodesConnectedTo(root, *edges)
			case unknown:
				fallthrough
			default:
				ids = findNodesConnectedTo(root, *edges)
			}

			for _, id := range ids {
				iterationNodeIds[id] = true
			}
		}

		for id := range iterationNodeIds {
			connectedNodeIds[id] = true
		}
	}

	for _, node := range *nodes {
		if _, ok := connectedNodeIds[node.Resource.id()]; !ok {
			removeID(node.Resource.id(), edges, nodes)
		}
	}
}

func findOutboundNodesConnectedTo(id string, edges []Edge) []string {
	nodeIdsMap := map[string]bool{}
	nodeIds := []string{}

	for _, edge := range edges {
		if edge.Source.id() == id {
			if _, ok := nodeIdsMap[edge.Destination.id()]; !ok {
				nodeIdsMap[edge.Destination.id()] = true
			}
		}
	}

	for k := range nodeIdsMap {
		nodeIds = append(nodeIds, k)
	}

	return nodeIds
}

func findInboundNodesConnectedTo(id string, edges []Edge) []string {
	nodeIdsMap := map[string]bool{}
	nodeIds := []string{}

	for _, edge := range edges {
		if edge.Destination.id() == id {
			if _, ok := nodeIdsMap[edge.Source.id()]; !ok {
				nodeIdsMap[edge.Source.id()] = true
			}
		}
	}

	for k := range nodeIdsMap {
		nodeIds = append(nodeIds, k)
	}

	return nodeIds
}

func findNodesConnectedTo(id string, edges []Edge) []string {
	nodeIdsMap := map[string]bool{}
	nodeIds := []string{}

	for _, edge := range edges {
		if edge.Source.id() == id {
			if _, ok := nodeIdsMap[edge.Destination.id()]; !ok {
				nodeIdsMap[edge.Destination.id()] = true
			}
		} else if edge.Destination.id() == id {
			if _, ok := nodeIdsMap[edge.Source.id()]; !ok {
				nodeIdsMap[edge.Source.id()] = true
			}
		}
	}

	for k := range nodeIdsMap {
		nodeIds = append(nodeIds, k)
	}

	return nodeIds
}
