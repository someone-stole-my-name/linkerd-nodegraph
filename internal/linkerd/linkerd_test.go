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
	t.Parallel()

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			edges, _ := edgesFromVec(testCase.prometheusEdgesResponse)
			nodes, _ := nodesFromVec(testCase.prometheusNodesResponse)
			stats := linkerd.Stats{mockGraphSource{edges, nodes}}

			graph, err := stats.Graph(context.Background(), testCase.graphParams)
			if err != nil {
				log.Fatal(err)
			}

			assert.Equal(t, &testCase.graphExpect, graph)
		})
	}
}
