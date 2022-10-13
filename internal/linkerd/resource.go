package linkerd

import (
	"fmt"

	"github.com/prometheus/common/model"
)

type resourceType int

const (
	DeploymentResourceType = resourceType(iota)
	StatefulsetResourceType
)

var (
	resourceTypes = []resourceType{
		DeploymentResourceType,
		StatefulsetResourceType,
	}
)

type Resource struct {
	Name         model.LabelValue
	Namespace    model.LabelValue
	ResourceType resourceType
}

func (rt resourceType) String() string {
	switch rt {
	case DeploymentResourceType:
		return "deployment"
	case StatefulsetResourceType:
		return "statefulset"
	}

	return "unknown"
}

func (rt resourceType) Label() model.LabelName {
	switch rt {
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
