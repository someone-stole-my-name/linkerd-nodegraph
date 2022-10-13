package linkerd

import (
	"fmt"
	"linkerd-nodegraph/internal/nodegraph"
)

type Edge struct {
	Source      Resource
	Destination Resource
}

func (e Edge) nodegraphEdge() nodegraph.Edge {
	return nodegraph.Edge{
		"id":     fmt.Sprintf("%s__%s", e.Source.id(), e.Destination.id()),
		"source": e.Source.id(),
		"target": e.Destination.id(),
	}
}
