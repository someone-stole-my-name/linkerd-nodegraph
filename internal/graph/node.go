package graph

import "fmt"

type Node struct {
	Name        string
	Namespace   string
	Type        resourceType
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
