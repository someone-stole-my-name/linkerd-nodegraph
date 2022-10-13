package nodegraph

import (
	"encoding/json"
	"errors"
)

var ErrInvalidGraphItem = errors.New("item does not conforms to spec")

type FieldType int

const (
	FieldTypeString FieldType = iota
	FieldTypeNumber
)

type Field struct {
	Name        string    `json:"field_name"`
	Type        FieldType `json:"type"`
	Color       string    `json:"color,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
}

type NodeFields struct {
	Node []Field `json:"nodes_fields"`
	Edge []Field `json:"edges_fields"`
}

type (
	Node map[string]interface{}
	Edge map[string]interface{}
)

type Graph struct {
	Spec  NodeFields `json:"-"`
	Nodes []Node     `json:"nodes"`
	Edges []Edge     `json:"edges"`
}

func (t FieldType) String() string {
	switch t {
	case FieldTypeString:
		return "string"
	case FieldTypeNumber:
		return "number"
	}

	return "unknown"
}

func (f FieldType) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (g *Graph) AddNode(n ...Node) error {
	for _, node := range n {
		if !validItem(node, g.Spec.Node) {
			return ErrInvalidGraphItem
		}
	}

	g.Nodes = append(g.Nodes, n...)

	return nil
}

func (g *Graph) AddEdge(e ...Edge) error {
	for _, edge := range e {
		if !validItem(edge, g.Spec.Edge) {
			return ErrInvalidGraphItem
		}
	}

	g.Edges = append(g.Edges, e...)

	return nil
}

func validItem(item map[string]interface{}, fields []Field) bool {
	for _, field := range fields {
		if _, ok := item[field.Name]; !ok {
			return false
		}

		switch field.Type {
		case FieldTypeString:
			if _, ok := item[field.Name].(string); !ok {
				return false
			}
		case FieldTypeNumber:
			if _, ok := item[field.Name].(int); ok {
				break
			}

			if _, ok := item[field.Name].(int32); ok {
				break
			}

			if _, ok := item[field.Name].(float32); ok {
				break
			}

			if _, ok := item[field.Name].(float64); ok {
				break
			}

			return false
		}
	}

	return true
}
