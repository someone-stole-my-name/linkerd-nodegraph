package linkerd_test

import (
	"context"
	"linkerd-nodegraph/internal/linkerd"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockGraphSource struct {
	edges *[]linkerd.Edge
	nodes *[]linkerd.Node
}

func (m mockGraphSource) Nodes(ctx context.Context) (*[]linkerd.Node, error) {
	return m.nodes, nil
}

func (m mockGraphSource) Edges(ctx context.Context) (*[]linkerd.Edge, error) {
	return m.edges, nil
}

func Test_Graph(t *testing.T) {
	for _, tt := range testCases {
		edges, _ := edgesFromVec(tt.prometheusEdgesResponse)
		nodes, _ := nodesFromVec(tt.prometheusNodesResponse)
		stats := linkerd.Stats{mockGraphSource{edges, nodes}}

		graph, err := stats.Graph(context.Background(), tt.graphParams)
		if err != nil {
			log.Fatal(err)
		}

		assert.Equal(t, &tt.graphExpect, graph)
	}
}
