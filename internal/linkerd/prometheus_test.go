package linkerd_test

import (
	"context"
	"linkerd-nodegraph/internal/linkerd"
	"log"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func edgesFromVec(v []model.Vector) (*[]linkerd.Edge, error) {
	return linkerd.PromGraphSource{
		Client: &outputPromMock{output: v},
	}.Edges(context.TODO())
}

func nodesFromVec(v []model.Vector) (*[]linkerd.Node, error) {
	return linkerd.PromGraphSource{
		Client: &outputPromMock{output: v},
	}.Nodes(context.TODO())
}

func Test_Edges(t *testing.T) {
	t.Parallel()

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			edges, err := edgesFromVec(testCase.prometheusEdgesResponse)
			if err != nil {
				log.Fatal(err)
			}

			assert.Equal(t, testCase.edgesExpect, edges)
		})
	}
}

func Test_Nodes(t *testing.T) {
	t.Parallel()

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			nodes, err := nodesFromVec(testCase.prometheusNodesResponse)
			if err != nil {
				log.Fatal(err)
			}

			assert.Equal(t, testCase.nodesExpect, nodes)
		})
	}
}
