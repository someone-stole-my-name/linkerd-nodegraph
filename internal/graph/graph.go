package graph

import "fmt"

type ResourceKind int

const (
	DeploymentKind ResourceKind = iota
	StatefulsetKind
	UndefinedKind
)

var Kinds = []ResourceKind{
	DeploymentKind,
	StatefulsetKind,
}

func (k ResourceKind) String() string {
	switch k {
	case DeploymentKind:
		return "deployment"
	case StatefulsetKind:
		return "statefulset"
	case UndefinedKind:
		fallthrough
	default:
		return "unknown"
	}
}

func ResourceKindFromString(k string) ResourceKind {
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
	Kind      ResourceKind
}

type Node struct {
	Resource Resource

	SuccessRate   float64
	LatencyP95    float64
	RequestVolume float64
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
