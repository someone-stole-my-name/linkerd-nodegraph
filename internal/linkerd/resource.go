package linkerd

import (
	"fmt"

	"github.com/prometheus/common/model"
)

type resourceType int

const (
	deploymentResourceType = resourceType(iota)
	statefulsetResourceType
)

var (
	resourceTypes = []resourceType{
		deploymentResourceType,
		statefulsetResourceType,
	}
)

type resource struct {
	name         model.LabelValue
	namespace    model.LabelValue
	resourceType resourceType
}

func (rt resourceType) String() string {
	switch rt {
	case deploymentResourceType:
		return "deployment"
	case statefulsetResourceType:
		return "statefulset"
	}
	return "unknown"
}

func (rt resourceType) Label() model.LabelName {
	switch rt {
	case deploymentResourceType:
		return deploymentLabel
	case statefulsetResourceType:
		return statefulsetLabel
	}
	return model.LabelName("unknown")
}

func (r resource) id() string {
	return fmt.Sprintf("%s__%s__%s", r.namespace, r.name, r.resourceType.String())
}
