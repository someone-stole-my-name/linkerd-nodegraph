package nodegraph_test

import (
	"encoding/json"
	"linkerd-nodegraph/internal/nodegraph"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

var fields = nodegraph.NodeFields{
	Edge: []nodegraph.Field{
		{Name: "foo", Type: nodegraph.FieldTypeString},
		{Name: "bar", Type: nodegraph.FieldTypeNumber},
	},
	Node: []nodegraph.Field{
		{Name: "foo", Type: nodegraph.FieldTypeString},
		{Name: "bar", Type: nodegraph.FieldTypeNumber},
		{
			Name:        "arc__foo",
			Type:        nodegraph.FieldTypeNumber,
			Color:       "foo",
			DisplayName: "foo",
		},
		{
			Name:        "arc__bar",
			Type:        nodegraph.FieldTypeString,
			Color:       "bar",
			DisplayName: "bar",
		},
	},
}

func Test_NodeFieldsMarshall(t *testing.T) {
	const expected = `{"nodes_fields":[{"field_name":"foo","type":"string"},{"field_name":"bar","type":"number"},{"field_name":"arc__foo","type":"number","color":"foo","displayName":"foo"},{"field_name":"arc__bar","type":"string","color":"bar","displayName":"bar"}],"edges_fields":[{"field_name":"foo","type":"string"},{"field_name":"bar","type":"number"}]}`
	b, err := json.Marshal(fields)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, expected, string(b))
}

func Test_GraphAdd(t *testing.T) {
	const expected = `{"nodes":[{"arc__bar":"baz","arc__foo":0.2,"bar":1,"foo":"bar"}],"edges":[{"bar":1,"foo":"bar"}]}`
	var g = nodegraph.Graph{Spec: fields}
	assert.Nil(t,
		g.AddEdge(map[string]interface{}{
			"foo": "bar",
			"bar": 1,
		}))
	assert.Equal(t, nodegraph.ErrInvalidGraphItem,
		g.AddEdge(map[string]interface{}{
			"foo": "bar",
			"bar": "1", // The spec defines bar as a number
		}))
	assert.Nil(t,
		g.AddNode(map[string]interface{}{
			"foo":      "bar",
			"bar":      1,
			"arc__foo": 0.2,
			"arc__bar": "baz",
		}))
	assert.Equal(t, nodegraph.ErrInvalidGraphItem,
		g.AddNode(map[string]interface{}{
			"foo":      "bar",
			"bar":      1,
			"arc__foo": "0.2", // The spec defines arc__foo as a number
			"arc__bar": "baz",
		}))
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(b))
}
