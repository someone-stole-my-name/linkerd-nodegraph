package linkerd

import (
	"context"
	"linkerd-nodegraph/internal/nodegraph"

	"github.com/prometheus/common/model"
)

type GraphSource interface {
	Query(ctx context.Context, query string) (model.Vector, error)
}

var (
	GraphSpec = nodegraph.NodeFields{
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

	namespaceLabel      = model.LabelName("namespace")
	dstNamespaceLabel   = model.LabelName("dst_namespace")
	deploymentLabel     = model.LabelName("deployment")
	statefulsetLabel    = model.LabelName("statefulset")
	dstDeploymentLabel  = model.LabelName("dst_deployment")
	dstStatefulsetLabel = model.LabelName("dst_statefulset")
)

type Stats struct {
	Server GraphSource
}

func (m Stats) Graph(ctx context.Context) (*nodegraph.Graph, error) {
	graph := nodegraph.Graph{Spec: GraphSpec}

	nodes, err := m.nodes(ctx)
	if err != nil {
		return nil, err
	}

	edges, err := m.edges(ctx)
	if err != nil {
		return nil, err
	}

	seenIds := map[string]bool{}
	for _, node := range nodes {
		nodegraphNode := node.nodegraphNode()
		err := graph.AddNode(nodegraphNode)
		if err != nil {
			return nil, err
		}
		seenIds[node.resource.id()] = true
	}

	for _, edge := range edges {
		nographEdge := edge.nodegraphEdge()
		err := graph.AddEdge(nographEdge)
		if err != nil {
			return nil, err
		}

		// if we don't have a node for the source/destination (ie not meshed stuff)
		// add a node for it
		for _, resource := range []resource{edge.source, edge.destination} {
			if _, ok := seenIds[resource.id()]; !ok {
				err := graph.AddNode(node{resource: resource}.nodegraphNode())
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return &graph, nil
}
