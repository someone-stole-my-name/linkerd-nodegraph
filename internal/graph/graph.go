package graph

import "fmt"

type kind int

const (
	DeploymentKind kind = iota
	StatefulsetKind
	UndefinedKind
)

var Kinds = []kind{
	DeploymentKind,
	StatefulsetKind,
}

func (k kind) String() string {
	switch k {
	case DeploymentKind:
		return "deployment"
	case StatefulsetKind:
		return "statefulset"
	}

	return "unknown"
}

func KindFromString(k string) kind {
	switch k {
	case "deployment":
		return DeploymentKind
	case "statefulset":
		return StatefulsetKind
	default:
		return UndefinedKind
	}
}

type Resource struct {
	Name      string
	Namespace string
	Kind      kind
}

type Node struct {
	Resource    Resource
	SuccessRate *float64
}

type Edge struct {
	Source      *Node
	Destination *Node
}

func (n Node) Id() string {
	return fmt.Sprintf("%s__%s__%s", n.Resource.Namespace, n.Resource.Name, n.Resource.Kind.String())
}

func (e Edge) Id() string {
	return fmt.Sprintf("%s__%s", e.Source.Id(), e.Destination.Id())
}
