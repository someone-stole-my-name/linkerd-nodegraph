package linkerd

import (
	"fmt"
	"linkerd-nodegraph/internal/nodegraph"
)

type Node struct {
	Resource    Resource
	SuccessRate *float64
}

func (n Node) success() float64 {
	if n.SuccessRate != nil {
		return *n.SuccessRate
	}

	return 0
}

func (n Node) failed() float64 {
	if n.SuccessRate != nil {
		return 1 - *n.SuccessRate
	}

	return 1
}

func (n Node) percent() string {
	if n.SuccessRate != nil {
		return fmt.Sprintf("%.2f%%", *n.SuccessRate*100) //nolint:gomnd
	}

	return "N/A"
}

func (n Node) nodegraphNode() nodegraph.Node {
	return nodegraph.Node{
		"id":                n.Resource.id(),
		"title":             fmt.Sprintf("%s/%s", n.Resource.Namespace, n.Resource.Name),
		"arc__failed":       n.failed(),
		"arc__success":      n.success(),
		"detail__type":      n.Resource.ResourceType.String(),
		"detail__namespace": string(n.Resource.Namespace),
		"detail__name":      string(n.Resource.Name),
		"mainStat":          n.percent(),
	}
}
