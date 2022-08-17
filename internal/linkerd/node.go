package linkerd

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/nodegraph"
)

type node struct {
	resource    resource
	successRate *float64
}

func (n node) success() float64 {
	if n.successRate != nil {
		return *n.successRate
	}
	return 0
}

func (n node) failed() float64 {
	if n.successRate != nil {
		return 1 - *n.successRate
	}
	return 1
}

func (n node) percent() string {
	if n.successRate != nil {
		return fmt.Sprintf("%.2f%%", *n.successRate*100)
	}
	return "N/A"
}

func (n node) nodegraphNode() nodegraph.Node {
	return nodegraph.Node{
		"id":                n.resource.id(),
		"title":             fmt.Sprintf("%s/%s", n.resource.namespace, n.resource.name),
		"arc__failed":       n.failed(),
		"arc__success":      n.success(),
		"detail__type":      n.resource.resourceType.String(),
		"detail__namespace": string(n.resource.namespace),
		"detail__name":      string(n.resource.name),
		"mainStat":          n.percent(),
	}
}

func (m Stats) nodes(ctx context.Context) ([]node, error) {
	nodes := []node{}
	queryFormat := `sum(irate(response_total{classification="success", direction="inbound", %[1]s!="", namespace!=""}[5m])) by (namespace, %[1]s) / sum(irate(response_total{direction="inbound", %[1]s!="", namespace!=""}[5m])) by (namespace, %[1]s) >= 0`
	for _, resourceType := range resourceTypes {
		vector, err := m.Server.Query(ctx, fmt.Sprintf(queryFormat, resourceType.String()))
		if err != nil {
			return nil, err
		}
		for _, v := range vector {
			value := float64(v.Value)
			nodes = append(nodes, node{
				successRate: &value,
				resource: resource{
					namespace:    v.Metric[namespaceLabel],
					resourceType: resourceType,
					name:         v.Metric[resourceType.Label()],
				},
			})
		}
	}
	return nodes, nil
}
