package linkerd_test

import (
	"context"
	"linkerd-nodegraph/internal/linkerd"
	"log"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func edgesFromVec(v []model.Vector) ([]linkerd.Edge, error) {
	return linkerd.PromGraphSource{
		Client: &outputPromMock{output: v},
	}.Edges(context.TODO())
}

func nodesFromVec(v []model.Vector) ([]linkerd.Node, error) {
	return linkerd.PromGraphSource{
		Client: &outputPromMock{output: v},
	}.Nodes(context.TODO())
}

func Test_Edges(t *testing.T) {
	for _, tt := range testCases {
		edges, err := edgesFromVec(tt.prometheusEdgesResponse)
		if err != nil {
			log.Fatal(err)
		}

		assert.Equal(t, tt.edgesExpect, edges)
	}
}

func Test_Nodes(t *testing.T) {
	for _, tt := range testCases {
		nodes, err := nodesFromVec(tt.prometheusNodesResponse)
		if err != nil {
			log.Fatal(err)
		}

		assert.Equal(t, tt.nodesExpect, nodes)
	}
}
