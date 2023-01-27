package graph

import (
	"context"
	"fmt"

	"github.com/prometheus/common/model"
)

type resourceType int

const (
	DeploymentResourceType = resourceType(iota)
	StatefulsetResourceType
)

var ResourceTypes = []resourceType{
	DeploymentResourceType,
	StatefulsetResourceType,
}

type graphSource interface {
	Node(resource Resource, ctx context.Context) (Node, error)
	Edges(ctx context.Context) (*[]Edge, error)
}

type Resource struct {
	Name      string
	Namespace string
	Type      resourceType
	Source    graphSource
}

func (t resourceType) String() string {
	switch t {
	case DeploymentResourceType:
		return "deployment"
	case StatefulsetResourceType:
		return "statefulset"
	}

	return "unknown"
}

func (t resourceType) Label() model.LabelName {
	switch t {
	case DeploymentResourceType:
		return deploymentLabel
	case StatefulsetResourceType:
		return statefulsetLabel
	}

	return model.LabelName("unknown")
}

func (r Resource) id() string {
	return fmt.Sprintf("%s__%s__%s", r.Namespace, r.Name, r.ResourceType.String())
}
