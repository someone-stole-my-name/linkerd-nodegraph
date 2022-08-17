package linkerd

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/nodegraph"

	"github.com/prometheus/common/model"
)

type edge struct {
	source      resource
	destination resource
}

func (e edge) nodegraphEdge() nodegraph.Edge {
	return nodegraph.Edge{
		"id":     fmt.Sprintf("%s__%s", e.source.id(), e.destination.id()),
		"source": e.source.id(),
		"target": e.destination.id(),
	}
}

func (m Stats) edges(ctx context.Context) ([]edge, error) {
	e := []edge{}
	vector, err := m.Server.Query(ctx, "sum(rate(response_total[5m])) by (deployment, statefulset, namespace, dst_namespace, dst_deployment, dst_statefulset)")
	if err != nil {
		return nil, err
	}

	for _, v := range vector {
		edge := parseSample(v)
		// some series are missing fields
		if edge != nil {
			e = append(e, *edge)
		}
	}

	return e, nil
}

func parseSample(s *model.Sample) *edge {
	e := edge{}

	if v, ok := s.Metric[deploymentLabel]; ok {
		e.source.resourceType = deploymentResourceType
		e.source.name = v
	} else if v, ok := s.Metric[statefulsetLabel]; ok {
		e.source.resourceType = statefulsetResourceType
		e.source.name = v
	} else {
		return nil
	}

	if v, ok := s.Metric[dstDeploymentLabel]; ok {
		e.destination.resourceType = deploymentResourceType
		e.destination.name = v
	} else if v, ok := s.Metric[dstStatefulsetLabel]; ok {
		e.destination.resourceType = statefulsetResourceType
		e.destination.name = v
	} else {
		return nil
	}

	if _, ok := s.Metric[namespaceLabel]; !ok {
		return nil
	}

	if _, ok := s.Metric[dstNamespaceLabel]; !ok {
		return nil
	}

	e.source.namespace = s.Metric[namespaceLabel]
	e.destination.namespace = s.Metric[dstNamespaceLabel]
	return &e
}
